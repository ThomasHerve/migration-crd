/*
Package v1alpha1 contains API Schema definitions for the database v1alpha1 API group
+kubebuilder:object:generate=true
+kubebuilder:resource:path=pgproxies,scope=Namespaced
+kubebuilder:subresource:status
*/
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PgProxySpec defines the desired state of PgProxy
// +kubebuilder:printcolumn:name="Deployment",type=string,JSONPath=".spec.deploymentName"
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="ServiceStatus",type=string,JSONPath=".status.serviceStatus"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp"

type PgProxySpec struct {
	// Name of the application Deployment to proxy and migrate
	DeploymentName string `json:"deploymentName"`
	// Namespace where the Deployment resides and resources will be created
	Namespace string `json:"namespace"`

	// PostgreSQL settings
	// Name of the service, use it to tell the deployment where to find the DB
	ProxyServiceName string `json:"proxyServiceName"`
	// Host or service name of the production PostgreSQL
	PostgresServiceHost string `json:"postgresServiceHost"`
	// Port of the production PostgreSQL
	PostgresServicePort int32 `json:"postgresServicePort"`

	// Migration settings (optional)
	// Secret containing credentials to dump and restore the database
	DumpSecret string `json:"dumpSecret,omitempty"`
	// Image containing the migration logic
	MigrationContainerImage string `json:"migrationContainerImage,omitempty"`
}

// PgProxyPhase represents the lifecycle phase of the proxy/migration workflow
type PgProxyPhase string

const (
	// PhasePending: CRD created but no action taken yet
	PhasePending PgProxyPhase = "Pending"
	// PhaseProxying: Only proxy is active
	PhaseProxying PgProxyPhase = "Proxying"
	// PhasePreparing: Temp database provisioning and restore in progress
	PhasePreparing PgProxyPhase = "Preparing"
	// PhaseSwitching: Traffic switched to temporary database
	PhaseSwitching PgProxyPhase = "Switching"
	// PhaseMigrating: Running migrations on real database
	PhaseMigrating PgProxyPhase = "Migrating"
	// PhaseBuffering: Writes to temp DB are being buffered
	PhaseBuffering PgProxyPhase = "Buffering"
	// PhaseReplaying: Replaying buffered writes to the migrated database
	PhaseReplaying PgProxyPhase = "Replaying"
	// PhaseFinalizing: Final traffic switch and cleanup
	PhaseFinalizing PgProxyPhase = "Finalizing"
	// PhaseCompleted: Workflow finished successfully
	PhaseCompleted PgProxyPhase = "Completed"
	// PhaseFailed: Workflow encountered an error
	PhaseFailed PgProxyPhase = "Failed"
)

type PgProxyServiceStatus string

const (
	// ServiceStatusEmpty: CRD created but no action taken yet
	ServiceStatusEmpty PgProxyServiceStatus = "Empty"
	// ServiceStatusNormal: The service point on the normal db
	ServiceStatusNormal PgProxyServiceStatus = "Normal"
	// ServiceStatusProxy: The service point on the proxy
	ServiceStatusProxy PgProxyServiceStatus = "Proxy"
)

// PgProxyStatus defines the observed state of PgProxy
// +kubebuilder:subresource:status

type PgProxyStatus struct {
	// Current phase of the workflow
	Phase PgProxyPhase `json:"phase,omitempty"`
	// Conditions represent detailed status conditions
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// ServiceName of the active proxy Service
	ServiceName string `json:"serviceName,omitempty"`
	// ServiceStatus represent in which state the service leading to the db is is
	ServiceStatus string `json:"serviceStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:object:subresource:status

// PgProxy is the Schema for the pgproxies API
// Combines proxy pass-through and migration workflow
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PgProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PgProxySpec   `json:"spec,omitempty"`
	Status PgProxyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PgProxyList contains a list of PgProxy
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PgProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PgProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PgProxy{}, &PgProxyList{})
}
