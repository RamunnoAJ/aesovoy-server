# Tarea: Regenerar Documentación Swagger

## Descripción
Tras la refactorización de rutas (Tarea 003), las anotaciones de Swagger en los handlers de la API quedaron desactualizadas, apuntando a rutas raíz (ej: `/products`) en lugar de las nuevas rutas API (ej: `/api/v1/products`).
Se solicitó:
1.  Actualizar las anotaciones `@Router` en los handlers.
2.  Regenerar los archivos de Swagger (`swagger.json`, `swagger.yaml`).

## Resultado
- Se actualizaron masivamente los comentarios `@Router /` a `@Router /api/v1/` en todos los archivos de `internal/api/*.go`.
- Se instaló la herramienta `swag` (v1.16.6).
- Se ejecutó `swag init -g main.go --output swagger` exitosamente.
- La documentación ahora refleja correctamente la nueva estructura de endpoints bajo `/api/v1`.

## Mejoras Futuras
- Agregar validación de Swagger en el pipeline de CI/CD (si existiera) para evitar desincronizaciones futuras.
- Revisar si los modelos de request/response en Swagger están completos (algunos warnings de tipos podrían haber aparecido, aunque en este run fue limpio).
