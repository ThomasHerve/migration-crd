import { NestFactory } from '@nestjs/core';
import { SwaggerModule, DocumentBuilder } from '@nestjs/swagger';
import { AppModule } from './app.module';

async function bootstrap() {
  const app = await NestFactory.create(AppModule,  { cors: true });
    const config = new DocumentBuilder()
    .setTitle('migration crd backend api')
    .setDescription('The migration crd backend API')
    .setVersion('1.0')
    .addTag('migration-crd')
    .build();
  const document = SwaggerModule.createDocument(app, config);
  SwaggerModule.setup('api', app, document);

  await app.listen(3000);
}
bootstrap();
