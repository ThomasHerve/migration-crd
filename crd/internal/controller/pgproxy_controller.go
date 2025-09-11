package controller

import (
	"context"
	"encoding/json"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
	"strings"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	databasev1alpha1 "github.com/ThomasHerve/migration-crd/api/v1alpha1"
)

const (
	AnnotationTrigger = "database.thomas-herve.fr/migrate"
	// annotation key where we store the original Service spec (JSON) for restoration
	AnnotationSavedServiceSpec = "database.thomas-herve.fr/saved-service-spec"
)

// PgProxyReconciler reconciles a PgProxy object
type PgProxyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Helpers (skeletons) -------------------------------------------------------

func (r *PgProxyReconciler) createBaseService(ctx context.Context, proxy *databasev1alpha1.PgProxy) error {
	if svc == nil {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tempSvcName,
				Namespace: proxy.Spec.Namespace,
				Labels:    map[string]string{"pgproxy": proxy.Name},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{{
					Port:       proxy.Spec.PostgresServicePort,
					TargetPort: intstr.FromInt(int(proxy.Spec.PostgresServicePort)),
				}},
				Selector: map[string]string{"pgproxy-temp": proxy.Name},
			},
		}
		if err := ctrl.SetControllerReference(proxy, svc, r.Scheme); err != nil {
			return err
		}
		if err := r.Create(ctx, svc); err != nil && !apierrors.IsAlreadyExists(err) {
			return err
		}
	} else {
		// Patch to base
	}
	return nil
}

func (r *PgProxyReconciler) ensureTempDBAndRestore(ctx context.Context, proxy *databasev1alpha1.PgProxy) error {
	// 1) create a StatefulSet for a temporary Postgres instance
	// 2) create a Service to expose it (temp-<name>-svc)
	// 3) create a Job that does pg_dump from real DB and pg_restore into temp DB
	// 4) wait for restore Job to complete
	// NOTE: credentials read from proxy.Spec.DumpSecret
	// TODO: make images/configurable via spec
	logger := log.FromContext(ctx)

	tempSvcName := fmt.Sprintf("%s-temp-db-svc", proxy.Name)
	stsName := fmt.Sprintf("%s-temp-db", proxy.Name)

	// 1) create Service for db (ClusterIP)
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tempSvcName,
			Namespace: proxy.Spec.Namespace,
			Labels:    map[string]string{"pgproxy": proxy.Name},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Port:       proxy.Spec.PostgresServicePort,
				TargetPort: intstr.FromInt(int(proxy.Spec.PostgresServicePort)),
			}},
			Selector: map[string]string{"pgproxy-temp": proxy.Name},
		},
	}

	if err := ctrl.SetControllerReference(proxy, svc, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, svc); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	// 2) create StatefulSet for postgres (very simple)
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stsName,
			Namespace: proxy.Spec.Namespace,
			Labels:    map[string]string{"pgproxy-temp": proxy.Name},
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: tempSvcName,
			Replicas:    int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"pgproxy-temp": proxy.Name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"pgproxy-temp": proxy.Name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "postgres",
						Image: "postgres:15", // configurable later
						Env: []corev1.EnvVar{
							{Name: "POSTGRES_PASSWORD", Value: "changeme"}, // better: from secret
						},
						Ports: []corev1.ContainerPort{{ContainerPort: int32(proxy.Spec.PostgresServicePort)}},
					}},
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(proxy, sts, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, sts); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	// 3) create Job to restore dump into temp DB
	// The Job image should contain pg_dump/pg_restore or psql. Use proxy.Spec.DumpSecret for credentials.
	jobName := fmt.Sprintf("%s-restore-job", proxy.Name)
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: proxy.Spec.Namespace},
		Spec: batchv1.JobSpec{
			BackoffLimit: int32Ptr(3),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:  "restore",
						Image: "postgres:15", // using postgres image for pg_dump/pg_restore
						// Command: should run pg_dump from real DB and pg_restore to temp DB
						// TODO: read creds from secret and build command
					}},
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(proxy, job, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, job); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	// 4) wait for Job completion (simple poll)
	logger.Info("waiting for restore job to complete", "job", jobName)
	var j batchv1.Job
	for i := 0; i < 120; i++ { // wait up to ~2 minutes (adjust)
		if err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: proxy.Spec.Namespace}, &j); err != nil {
			return err
		}
		if j.Status.Succeeded > 0 {
			logger.Info("restore job succeeded")
			return nil
		}
		if j.Status.Failed > 0 && *j.Spec.BackoffLimit <= j.Status.Failed {
			return fmt.Errorf("restore job failed")
		}
		time.Sleep(1 * time.Second)
	}
	return fmt.Errorf("timeout waiting for restore job")
}

func (r *PgProxyReconciler) switchTrafficToTempAndTriggerNewPods(ctx context.Context, proxy *databasev1alpha1.PgProxy, deploy *appsv1.Deployment) error {
	// Steps:
	// 1) find the Service that application uses (proxy.Spec.PostgresServiceHost)
	//    -> support forms: "name", "name.namespace.svc.cluster.local", etc.
	// 2) fetch that Service, save its spec into an annotation on that Service (json)
	// 3) patch that Service to be type ExternalName with ExternalName=tempSvcFQDN (temp service)
	// 4) patch the Deployment template to set an env var that forces new pods to use the real DB (this triggers rolling update)
	//    e.g. set APP_DB_TARGET=real (your app must honor it)
	logger := log.FromContext(ctx)

	// resolve service name and ns
	svcName, svcNamespace := parseServiceHost(proxy.Spec.PostgresServiceHost, proxy.Spec.Namespace)
	origSvc := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: svcNamespace}, origSvc); err != nil {
		return err
	}

	// store original spec
	origSpecBytes, _ := json.Marshal(origSvc.Spec)
	if origSvc.Annotations == nil {
		origSvc.Annotations = map[string]string{}
	}
	origSvc.Annotations[AnnotationSavedServiceSpec] = string(origSpecBytes)

	// compute temp service FQDN: <proxy.Name>-temp-db-svc.<ns>.svc.cluster.local
	tempFQDN := fmt.Sprintf("%s-temp-db-svc.%s.svc.cluster.local", proxy.Name, proxy.Spec.Namespace)

	// patch to ExternalName
	origSvc.Spec = corev1.ServiceSpec{
		Type:         corev1.ServiceTypeExternalName,
		ExternalName: tempFQDN,
		Ports: []corev1.ServicePort{{
			Port:       proxy.Spec.PostgresServicePort,
			TargetPort: intstr.FromInt(int(proxy.Spec.PostgresServicePort)),
		}},
	}

	if err := r.Update(ctx, origSvc); err != nil {
		return err
	}
	logger.Info("patched application DB Service to ExternalName -> temp DB", "service", svcName, "temp", tempFQDN)

	// 4) patch deployment Pod template to add an env var to force new pods to use real DB
	//    we add or update env var "PGDB_FORCE_REAL" with timestamp so podTemplateHash changes and triggers rolling update
	updated := deploy.DeepCopy()
	containers := updated.Spec.Template.Spec.Containers
	for i := range containers {
		containers[i].Env = upsertEnv(containers[i].Env, corev1.EnvVar{
			Name:  "PGDB_FORCE_REAL",
			Value: fmt.Sprintf("true-%d", time.Now().Unix()),
		})
	}
	if err := r.Update(ctx, updated); err != nil {
		return err
	}
	logger.Info("patched Deployment to force new pods (that should connect to real DB)", "deployment", deploy.Name)

	return nil
}

func (r *PgProxyReconciler) checkMigrationCompleted(ctx context.Context, proxy *databasev1alpha1.PgProxy) (bool, error) {
	// This is application-specific. Options:
	// - The new pods create a Job/Signal when migration done
	// - The operator checks the logs of new pods or watches for a Condition
	// For this skeleton, assume user created a Job named <proxy.Name>-migration-done
	jobName := fmt.Sprintf("%s-migration-done", proxy.Name)
	var job batchv1.Job
	if err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: proxy.Spec.Namespace}, &job); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if job.Status.Succeeded > 0 {
		return true, nil
	}
	return false, nil
}

func (r *PgProxyReconciler) checkReadyToReplay(ctx context.Context, proxy *databasev1alpha1.PgProxy) (bool, error) {
	// Inspect the proxy buffer health (custom). For skeleton we just return true after a small wait.
	time.Sleep(1 * time.Second)
	return true, nil
}

func (r *PgProxyReconciler) replayBufferIntoRealDB(ctx context.Context, proxy *databasev1alpha1.PgProxy) error {
	// Create a Job that reads the buffer from the proxy's PVC (or object store) and replays it into real DB
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-replay-job", proxy.Name),
			Namespace: proxy.Spec.Namespace,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: int32Ptr(3),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{{
						Name:  "replay",
						Image: "your-registry/sql-replayer:latest", // TODO: implement image that reads buffer and applies it
					}},
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(proxy, job, r.Scheme); err != nil {
		return err
	}
	if err := r.Create(ctx, job); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	// wait for completion (simple poll)
	var j batchv1.Job
	for i := 0; i < 600; i++ {
		if err := r.Get(ctx, types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, &j); err != nil {
			return err
		}
		if j.Status.Succeeded > 0 {
			return nil
		}
		if j.Status.Failed > 0 && *j.Spec.BackoffLimit <= j.Status.Failed {
			return fmt.Errorf("replay job failed")
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for replay job")
}

func (r *PgProxyReconciler) restoreOriginalServiceAndCleanup(ctx context.Context, proxy *databasev1alpha1.PgProxy) error {
	// 1) find the original Service and its saved spec from annotation, restore it
	// 2) delete temp statefulset/service/jobs/proxy resources
	logger := log.FromContext(ctx)

	svcName, svcNamespace := parseServiceHost(proxy.Spec.PostgresServiceHost, proxy.Spec.Namespace)
	svc := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: svcName, Namespace: svcNamespace}, svc); err != nil {
		return err
	}
	if saved, ok := svc.Annotations[AnnotationSavedServiceSpec]; ok {
		var origSpec corev1.ServiceSpec
		if err := json.Unmarshal([]byte(saved), &origSpec); err == nil {
			svc.Spec = origSpec
			// remove annotation
			delete(svc.Annotations, AnnotationSavedServiceSpec)
			if err := r.Update(ctx, svc); err != nil {
				return err
			}
			logger.Info("restored original Service spec", "service", svcName)
		}
	}

	// cleanup temp resources
	// delete statefulset
	stsName := fmt.Sprintf("%s-temp-db", proxy.Name)
	sts := &appsv1.StatefulSet{}
	_ = r.Get(ctx, types.NamespacedName{Name: stsName, Namespace: proxy.Spec.Namespace}, sts)
	_ = r.Delete(ctx, sts)

	// delete temp svc
	tempSvcName := fmt.Sprintf("%s-temp-db-svc", proxy.Name)
	tempSvc := &corev1.Service{}
	_ = r.Get(ctx, types.NamespacedName{Name: tempSvcName, Namespace: proxy.Spec.Namespace}, tempSvc)
	_ = r.Delete(ctx, tempSvc)

	// delete jobs (restore/replay)
	_ = r.cleanupJobByName(ctx, fmt.Sprintf("%s-restore-job", proxy.Name), proxy.Spec.Namespace)
	_ = r.cleanupJobByName(ctx, fmt.Sprintf("%s-replay-job", proxy.Name), proxy.Spec.Namespace)

	// TODO: delete proxy Deployment & PVCs

	return nil
}

func (r *PgProxyReconciler) cleanupJobByName(ctx context.Context, name, ns string) error {
	var j batchv1.Job
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, &j); err != nil {
		return nil
	}
	foreground := metav1.DeletePropagationForeground
	return r.Delete(ctx, &j, &client.DeleteOptions{PropagationPolicy: &foreground})
}

// utility helpers ----------------------------------------------------------

func int32Ptr(i int32) *int32 { return &i }

func parseServiceHost(host string, defaultNS string) (name, ns string) {
	// naive parser: supports "name" or "name.namespace.svc.cluster.local"
	// if host contains '.', take first two fields as name.namespace
	if host == "" {
		return "", defaultNS
	}
	parts := strings.Split(host, ".")
	if len(parts) == 1 {
		return parts[0], defaultNS
	}
	return parts[0], parts[1]
}

func upsertEnv(env []corev1.EnvVar, new corev1.EnvVar) []corev1.EnvVar {
	found := false
	for i := range env {
		if env[i].Name == new.Name {
			env[i].Value = new.Value
			found = true
			break
		}
	}
	if !found {
		env = append(env, new)
	}
	return env
}

func (r *PgProxyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasev1alpha1.PgProxy{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=database.thomas-herve.fr,resources=pgproxies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=database.thomas-herve.fr,resources=pgproxies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

func (r *PgProxyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var proxy databasev1alpha1.PgProxy
	if err := r.Get(ctx, req.NamespacedName, &proxy); err != nil {
		if apierrors.IsNotFound(err) {
			// ressource supprimée
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Find the Deployment referenced by this PgProxy
	if proxy.Spec.Namespace == "" {
		proxy.Spec.Namespace = req.Namespace
	}

	deploy := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: proxy.Spec.DeploymentName, Namespace: proxy.Spec.Namespace}, deploy); err != nil {
		logger.Error(err, "unable to get Deployment", "name", proxy.Spec.DeploymentName)
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Check if the deployment has the migrate annotation
	deployAnno := deploy.Annotations
	trigger := false
	if deployAnno != nil {
		if v, ok := deployAnno[AnnotationTrigger]; ok && (v == "true" || v == "1") {
			trigger = true
		}
	}
	/*
		logger.Info("--------------------")
		logger.Info(proxy.Status.Phase)
		logger.Info("--------------------")
	*/
	// If no trigger, ensure status is at least Pending/Proxying and bail
	if !trigger && proxy.Status.Phase == "" {
		proxy.Status.Phase = databasev1alpha1.PhasePending
		if err := createBaseService(ctx, &proxy); err != nil {
			logger.Error(err, "create base service failed")
			proxy.Status.Phase = databasev1alpha1.PhaseFailed
			_ = r.Status().Update(ctx, &proxy)
			return ctrl.Result{}, err
		}
		_ = r.Status().Update(ctx, &proxy)
		return ctrl.Result{}, nil
	}

	// state machine based on proxy.Status.Phase
	switch proxy.Status.Phase {
	case "":
		proxy.Status.Phase = databasev1alpha1.PhasePreparing
		_ = r.Status().Update(ctx, &proxy)
		return ctrl.Result{Requeue: true}, nil

	case databasev1alpha1.PhasePreparing:
		logger.Info("PhasePreparing: provisioning temp DB and restoring dump")
		if err := r.ensureTempDBAndRestore(ctx, &proxy); err != nil {
			logger.Error(err, "ensureTempDBAndRestore failed")
			proxy.Status.Phase = databasev1alpha1.PhaseFailed
			_ = r.Status().Update(ctx, &proxy)
			return ctrl.Result{}, err
		}
		proxy.Status.Phase = databasev1alpha1.PhaseSwitching
		_ = r.Status().Update(ctx, &proxy)
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil

	case databasev1alpha1.PhaseSwitching:
		logger.Info("PhaseSwitching: patching Service to point to temp DB and forcing new pods to use real DB")
		if err := r.switchTrafficToTempAndTriggerNewPods(ctx, &proxy, deploy); err != nil {
			logger.Error(err, "switchTraffic failed")
			proxy.Status.Phase = databasev1alpha1.PhaseFailed
			_ = r.Status().Update(ctx, &proxy)
			return ctrl.Result{}, err
		}
		proxy.Status.Phase = databasev1alpha1.PhaseMigrating
		_ = r.Status().Update(ctx, &proxy)
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil

	case databasev1alpha1.PhaseMigrating:
		logger.Info("PhaseMigrating: waiting for new pods to finish migration")
		// We need a signal that new pods have finished migration.
		// For this skeleton we look for a Job completion or a Deployment condition / custom signal.
		ok, err := r.checkMigrationCompleted(ctx, &proxy)
		if err != nil {
			logger.Error(err, "checkMigrationCompleted error")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		if !ok {
			// still migrating
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}

		// start buffering phase (proxy already captures SQL into a buffer)
		proxy.Status.Phase = databasev1alpha1.PhaseBuffering
		_ = r.Status().Update(ctx, &proxy)
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil

	case databasev1alpha1.PhaseBuffering:
		logger.Info("PhaseBuffering: ensure buffer is healthy / capturing")
		// Ideally check proxy buffer size or health
		// Transition to Replaying when migration done for real DB + buffer ready to replay
		readyToReplay, err := r.checkReadyToReplay(ctx, &proxy)
		if err != nil {
			logger.Error(err, "checkReadyToReplay")
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		if !readyToReplay {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		proxy.Status.Phase = databasev1alpha1.PhaseReplaying
		_ = r.Status().Update(ctx, &proxy)
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil

	case databasev1alpha1.PhaseReplaying:
		logger.Info("PhaseReplaying: replaying buffer into migrated DB")
		if err := r.replayBufferIntoRealDB(ctx, &proxy); err != nil {
			logger.Error(err, "replayBufferIntoRealDB failed")
			proxy.Status.Phase = databasev1alpha1.PhaseFailed
			_ = r.Status().Update(ctx, &proxy)
			return ctrl.Result{}, err
		}
		proxy.Status.Phase = databasev1alpha1.PhaseFinalizing
		_ = r.Status().Update(ctx, &proxy)
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil

	case databasev1alpha1.PhaseFinalizing:
		logger.Info("PhaseFinalizing: restore original Service and cleanup")
		if err := r.restoreOriginalServiceAndCleanup(ctx, &proxy); err != nil {
			logger.Error(err, "cleanup failed")
			proxy.Status.Phase = databasev1alpha1.PhaseFailed
			_ = r.Status().Update(ctx, &proxy)
			return ctrl.Result{}, err
		}
		proxy.Status.Phase = databasev1alpha1.PhaseCompleted
		_ = r.Status().Update(ctx, &proxy)
		return ctrl.Result{}, nil

	case databasev1alpha1.PhaseCompleted:
		// nothing to do
		return ctrl.Result{}, nil

	case databasev1alpha1.PhaseFailed:
		// noop, or try to rollback depending on policy
		return ctrl.Result{}, nil

	default:
		logger.Info("unknown phase", "phase", proxy.Status.Phase)
		return ctrl.Result{}, nil
	}
}
