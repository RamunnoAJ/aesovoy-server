# Recomendaciones de Seguridad para VPS

Aunque la aplicación tiene una arquitectura segura (Docker, Caddy, protección contra inyecciones SQL y CSRF), es fundamental asegurar el servidor (VPS) donde se aloja.

## 1. Configurar Firewall (UFW)
Protege el servidor bloqueando conexiones no deseadas. Solo deben estar abiertos los puertos para SSH, HTTP y HTTPS.

**Comandos para ejecutar en el VPS:**
```bash
# Permitir conexión SSH (puerto 22 por defecto)
sudo ufw allow ssh

# Permitir tráfico web
sudo ufw allow 80
sudo ufw allow 443

# Activar el firewall
sudo ufw enable
```

## 2. Proteger el Acceso SSH
El puerto 22 es el objetivo principal de ataques de fuerza bruta.

*   **Usar Claves SSH:** Configura el acceso mediante claves pública/privada y evita usar contraseñas.
*   **Deshabilitar acceso por contraseña:** Edita `/etc/ssh/sshd_config` y establece `PasswordAuthentication no`.
*   **Instalar Fail2Ban:** Herramienta que bloquea automáticamente IPs con múltiples intentos fallidos de login.
    ```bash
    sudo apt update
    sudo apt install fail2ban
    ```

## 3. Protección contra Fuerza Bruta (Rate Limiting)
Para evitar que intenten adivinar contraseñas de usuarios en la web masivamente:

*   **Caddy:** Puedes configurar límites de peticiones en el `Caddyfile`.
*   **Contraseñas Fuertes:** Asegúrate de utilizar contraseñas robustas para los usuarios administradores.

## 4. Backups (Copias de Seguridad)
La seguridad también implica recuperación ante desastres.

*   Configura una tarea automática (`cronjob`) que realice un volcado de la base de datos periódicamente.
*   Ejemplo de comando manual para backup (desde dentro del VPS):
    ```bash
    docker exec aesovoy_db pg_dump -U postgres aesovoy_prod > backup_$(date +%F).sql
    ```
*   **Importante:** Copia estos backups fuera del servidor regularmente.

## 5. Actualizaciones
Mantén el sistema operativo del VPS actualizado regularmente:
```bash
sudo apt update && sudo apt upgrade -y
```
