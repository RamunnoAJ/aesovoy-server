# Bitácora de Estructura y Arquitectura del Proyecto "A Eso Voy"

Este documento describe la arquitectura actual, el flujo de datos y las convenciones utilizadas en el proyecto. Sirve como referencia para el desarrollo continuo.

## 1. Stack Tecnológico

### Backend
- **Lenguaje:** Go (Golang).
- **Router:** `go-chi/chi` (v5).
- **Base de Datos:** PostgreSQL.
- **Driver SQL:** `lib/pq` (Acceso nativo con `database/sql`, sin ORM).
- **Manejo de Excel:** `xuri/excelize/v2`.
- **Inyección de Dependencias:** Manual (definida en `internal/app/app.go`).

### Frontend
- **Renderizado:** Server-Side Rendering (SSR) nativo de Go (`html/template`).
- **Interactividad:** Alpine.js (v3).
- **AJAX / SPA Feel:** HTMX.
- **Estilos:** Tailwind CSS (vía CDN).

## 2. Estructura de Directorios

```
/server
├── main.go                 # Punto de entrada. Inicializa App y arranca el servidor.
├── docker-compose.yaml     # Orquestación de contenedores (App + DB).
├── internal/               # Código privado de la aplicación.
│   ├── api/                # (Handlers) Controladores HTTP. Reciben requests y llaman a Stores/Services.
│   ├── app/                # (Wire) Inicialización de dependencias, base de datos y configuración global.
│   ├── billing/            # Lógica de generación de facturas/remitos (Excel).
│   ├── middleware/         # Auth, Logging, CSRF, Security Headers.
│   ├── routes/             # Definición de rutas y agrupación por roles (Admin/User).
│   ├── services/           # Lógica de negocio compleja (ej. LocalSale, Stocks).
│   ├── store/              # (Repository Pattern) Acceso a datos. Queries SQL crudas.
│   ├── views/              # Lógica de renderizado.
│   │   ├── renderer.go     # Configuración de templates y FuncMap (ej. jsToJson, formatMoney).
│   │   └── templates/      # Archivos .html (base.html, formularios, listados).
│   └── utils/              # Funciones de ayuda generales (Respuestas JSON, errores).
├── migrations/             # Scripts SQL para la estructura de la BD.
├── docs/                   # Documentación y assets estáticos (logos, plantillas Excel).
└── facturas/               # Directorio donde se guardan los Excel generados.
```

## 3. Patrones de Arquitectura

### Flujo de Datos
1.  **Request:** Llega a `main.go`, pasa por el router (`internal/routes`).
2.  **Middleware:** Se verifica autenticación (`RequireUser`, `RequireAdmin`) y seguridad.
3.  **Handler (`internal/api`):**
    *   Parsea el request (Forms, JSON).
    *   Valida datos básicos.
    *   Llama al `Store` (para CRUD simple) o `Service` (para lógica compleja).
4.  **Store (`internal/store`):** Ejecuta SQL directo contra PostgreSQL y mapea resultados a Structs.
5.  **Response:**
    *   **HTML:** Usa `renderer.Render` para devolver vistas completas o parciales (HTMX).
    *   **JSON:** Usa `utils.JSONResponse` (principalmente para APIs internas o errores).

### Convenciones de Frontend

#### Base Layout (`base.html`)
*   Contiene la estructura HTML5, Navbar, Sidebar y Scripts globales.
*   Define el bloque `{{block "content" .}}{{end}}` donde se inyectan las vistas.
*   Maneja el modal de confirmación global y los "Toasts" de notificación.

#### Componentes Dinámicos (Alpine.js)
*   Utilizamos Alpine.js para lógica de UI local (toggles, modales simples, calculadoras en cliente).
*   **Patrón "ProductItemManager":** Una función reutilizable en `base.html` (`createProductItemManager`) maneja listas dinámicas de productos (filas de items) con:
    *   Búsqueda en tiempo real (Combobox).
    *   Cálculo de subtotales y totales.
    *   Selección de precios (Minorista vs Mayorista/Dietética).

#### JSON Injection
*   Para pasar datos complejos de Go a JS (ej. lista de productos para el buscador), usamos la función `jsToJson` en una etiqueta `<script>` global o dentro del componente, evitando problemas de escape de caracteres.
    ```html
    <script>
        window.productsData = JSON.parse({{jsToJson .Products}});
    </script>
    ```

## 4. Módulos Principales

*   **Productos:** Gestión de inventario, precios (Unitario/Distribución) y recetas.
*   **Órdenes:** Pedidos de clientes. Generan movimientos de stock y se pueden exportar a Excel (Facturas).
*   **Ventas Local:** Ventas directas en el local. Descuentan stock "Local".
*   **Billing:** Generación de archivos Excel basados en una plantilla (`docs/Plantilla.xlsx`).
    *   Nombramiento: `[NombreCliente].xlsx` (saneado y truncado a 31 chars).
*   **Usuarios/Auth:** Sistema de roles (`administrator`, `employee`). Tokens de sesión y Cookies.

## 5. Notas de Desarrollo

*   **Base de Datos:** Al no usar ORM, cualquier cambio en la estructura de la BD requiere:
    1.  Crear archivo `.sql` en `migrations/`.
    2.  Actualizar los structs en `internal/store`.
    3.  Actualizar las queries SQL en los métodos del Store.
*   **Formateo de Precios:** Se utiliza la función `formatMoney` en los templates para mostrar moneda formateada ($ 1.234,56).
*   **HTMX:** Se usa para interacciones sin recarga completa (ej. `hx-post`, `hx-target="body"`).
