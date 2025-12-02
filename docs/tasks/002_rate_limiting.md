# Tarea: Implementar Rate Limiting con Sliding Window

## Descripción
Se solicitó implementar un limitador de tasa (rate limiter) utilizando `go-chi/httprate` con la estrategia de "ventana deslizante" (sliding window).
Requisitos específicos:
1.  **Límite Global:** 100 peticiones por minuto por IP para todas las rutas.
2.  **Límite Login:** Restricción más severa para la ruta de inicio de sesión (se definió en 5 peticiones por minuto).

## Resultado
- Se agregó la dependencia `github.com/go-chi/httprate`.
- Se modificó `internal/routes/routes.go`:
    - Se aplicó el middleware global `httprate.Limit(100, 1*time.Minute, ...)` al router principal.
    - Se aplicó un middleware específico `httprate.Limit(5, 1*time.Minute, ...)` a la ruta `POST /login` usando `r.With(...)`.

## Mejoras Futuras
- Aplicar el límite estricto también a la ruta de API `POST /tokens/authentication`.
- Configurar respuestas personalizadas (HTTP 429) con mensajes amigables o JSON según el contexto (API vs Web).
- Externalizar los valores de configuración (100 y 5) a variables de entorno para no tener "magic numbers" en el código.
