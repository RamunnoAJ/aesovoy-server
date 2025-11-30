# Guía para Contratar un VPS en Argentina

Para desplegar la aplicación, necesitas un Servidor Privado Virtual (VPS). Al estar en Argentina, tienes dos estrategias principales: priorizar el pago local/baja latencia o priorizar tecnología/estabilidad internacional.

## Opción 1: Proveedores Locales (Pago en Pesos + Latencia Mínima)
Ideal si buscas factura local (A/B) y métodos de pago sencillos (MercadoPago, Transferencia). La latencia es mínima (<15ms).

1.  **DonWeb (Cloud Server):**
    *   **Recomendado para iniciar.** Muy popular y fácil de contratar.
    *   *Atención:* Contratar "Cloud Server", NO "Hosting Compartido" (este último no sirve para Docker).
2.  **Huawei Cloud (Nodo Buenos Aires):**
    *   Infraestructura de nivel empresarial con presencia física en CABA.
    *   Panel de control más complejo, similar a AWS.
3.  **Wiroos / Baehost:**
    *   Proveedores de hosting tradicionales con buenas opciones de VPS y soporte local.

## Opción 2: Proveedores Internacionales (Tecnología Superior)
Si puedes pagar en dólares con tarjeta y asumir los impuestos (PAIS + Ganancias), suelen ofrecer paneles de control muy superiores, documentación extensa y backups automáticos sencillos.

*   **DigitalOcean, Vultr, Hetzner, Linode.**
*   *Nota:* La latencia suele ser de ~140ms (USA/Europa), lo cual es imperceptible para este tipo de aplicaciones de gestión.

---

## Requisitos Técnicos (Sizing)
Para correr la aplicación (`Go` + `Postgres` + `Docker`), busca estas especificaciones mínimas al contratar:

*   **Sistema Operativo:** Ubuntu 22.04 LTS o Debian 11/12 (Evitar Windows o CentOS).
*   **CPU:** 1 o 2 vCPU.
*   **RAM:** Mínimo 1 GB (Ideal **2 GB** para estar holgado con la base de datos).
*   **Disco:** 20 GB o 30 GB SSD es suficiente.

---

## Pasos para la Contratación y Acceso

1.  **Registro:** Crea la cuenta en el proveedor.
2.  **Selección:** Elige el plan "Cloud Server" o "VPS" con los recursos mencionados arriba.
3.  **Pago:** Completa el pago.
4.  **Datos de Acceso:** Recibirás un correo con la **IP Pública** (ej. `190.210.x.x`) y la contraseña del usuario `root`.

### Cómo conectarse
Una vez activo, abre tu terminal en Linux y escribe:

```bash
ssh root@IP_DE_TU_VPS
# Ejemplo: ssh root@190.210.10.5
```

Te pedirá la contraseña (que no se ve al escribirla). Una vez dentro, puedes proceder con la instalación de Docker y el despliegue usando el script `deploy.sh`.
