# Tarea: Refactorización de Rutas y Solución de Conflictos

## Descripción
Se detectó una duplicación crítica en las rutas (endpoints) donde los handlers de la API JSON y los handlers de las Vistas Web competían por las mismas URLs (ej: `GET /products`, `GET /orders`).
La tarea consistió en:
1.  Separar las rutas de la API bajo un prefijo `/api/v1`.
2.  Mantener las rutas de la Web en la raíz (`/`).
3.  Reubicar handlers mal posicionados (ej: `Invoices` y `Users` estaban mezclados).

## Resultado
- Se modificó `internal/routes/routes.go`.
- **Estructura Nueva:**
    - **API:** Todas las rutas JSON ahora viven bajo `/api/v1` (ej: `/api/v1/products`).
    - **Web:** Las rutas HTML permanecen en la raíz (ej: `/products`).
- Se corrigió la ubicación de `InvoiceHandler` y `UserHandler` (Web) para que estén explícitamente en el grupo de Vistas Web Protegidas.
- Se movieron los endpoints de registro y autenticación (`/users`, `/tokens/authentication`) dentro del grupo de API.

## Mejoras Futuras
- **Actualizar Swagger:** La documentación de Swagger (`swagger/`) probablemente referencia las rutas viejas (sin `/api/v1`). Deberá regenerarse.
- **Actualizar Frontend (si aplica):** Si hay código JS cliente (alpine.js) que llame a la API JSON, debe actualizarse para apuntar a `/api/v1/...`. (Nota: Actualmente la mayoría de las interacciones parecen ser HTMX/SSR, pero vale la pena revisar).
- **Tests:** Ejecutar los tests de integración para asegurar que no se rompieron los flujos de API.
