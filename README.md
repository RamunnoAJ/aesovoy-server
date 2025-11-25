# A Eso Voy - Backend

Este repositorio contiene el código fuente del servicio backend para la aplicación "A Eso Voy". Está construido en Go, siguiendo principios de arquitectura limpia y modular para facilitar su mantenimiento y escalabilidad.

## Prácticas y Arquitectura

El proyecto sigue una serie de buenas prácticas y patrones de diseño para asegurar un código robusto y mantenible.

### Estructura del Proyecto

La organización del código se centra en el directorio `internal`, que contiene la lógica principal de la aplicación, separada por funcionalidad:

-   `internal/api`: Contiene los manejadores (handlers) HTTP. Cada manejador es responsable de recibir las solicitudes, validarlas y comunicarse con la capa de datos.
-   `internal/store`: Es la capa de acceso a datos. Se encarga de todas las interacciones con la base de datos, abstrayendo la lógica de las consultas SQL del resto de la aplicación.
-   `internal/routes`: Define todas las rutas de la API y las asocia con sus respectivos manejadores. Se utiliza `chi` para el enrutamiento.
-   `internal/app`: Realiza la configuración inicial de la aplicación: conecta la base de datos, inicializa las dependencias y levanta el servidor.
-   `internal/middleware`: Contiene middlewares para funcionalidades transversales como la autenticación y el logging.
-   `migrations/`: Almacena los scripts de migración de la base de datos en archivos SQL.

### Patrones de Diseño

-   **Inyección de Dependencias:** Las dependencias se inyectan en el momento de la inicialización. Por ejemplo, los manejadores (`api`) reciben una instancia del `store`, lo que permite un bajo acoplamiento y facilita enormemente las pruebas unitarias al poder "mockear" la capa de datos.
-   **Repositorio (Capa `store`):** La capa `store` actúa como un repositorio, proveyendo una interfaz clara para acceder a los datos sin exponer los detalles de la implementación de la base de datos.

### Base de Datos

-   **Migraciones:** El esquema de la base de datos se gestiona a través de archivos SQL numerados en el directorio `migrations`. Esto permite un control de versiones de la base de datos y facilita la consistencia entre diferentes entornos.
-   **Consultas Nativas:** Se utiliza el paquete estándar `database/sql` para ejecutar consultas SQL nativas, lo que otorga un control total sobre el rendimiento y la lógica de las consultas.

### Testing

-   Se incluyen pruebas unitarias (`_test.go`) para la capa de `store`. Esto asegura que la lógica de acceso a datos es correcta y funciona como se espera.

## 🚀 Cómo Empezar

### Prerrequisitos

-   Docker
-   Docker Compose

### Pasos para la Instalación

1.  **Clonar el repositorio:**
    ```bash
    git clone https://github.com/RamunnoAJ/aesovoy-server.git
    cd server
    ```

2.  **Configurar el entorno:**
    Copia el archivo de ejemplo para las variables de entorno y ajústalo según tu configuración local.
    ```bash
    cp .env.example .env
    ```
    Asegúrate de rellenar las variables en el archivo `.env` (credenciales de la base de datos, secretos, etc.).

3.  **Levantar los servicios:**
    Usa Docker Compose para construir y levantar la aplicación y la base de datos.
    ```bash
    docker-compose up --build
    ```

4. **Ejecuta el script de inicialización**
   ```bash
   go run .
   ```

El servidor estará corriendo en el puerto especificado en tu archivo `.env`.
