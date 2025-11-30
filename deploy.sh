#!/bin/bash

# Detener el script si hay algún error
set -e

echo "Iniciando despliegue..."

# 1. Descargar últimos cambios
echo "Descargando cambios desde Git..."
git pull

# 2. Reconstruir y reiniciar contenedores (en segundo plano)
echo "Reconstruyendo y reiniciando contenedores..."
docker compose -f docker-compose.prod.yaml up -d --build

# 3. Limpiar imágenes antiguas para liberar espacio
echo "Limpiando imágenes antiguas de Docker..."
docker image prune -f

echo "¡Despliegue completado exitosamente!"
