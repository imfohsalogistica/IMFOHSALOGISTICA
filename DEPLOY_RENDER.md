# Render: entorno de pruebas

Runtime Go; build: `sh render-build.sh`; inicio: `./server_render`.
Variables: `IMFOHSA_DISABLE_TUNNEL=1`, `IMFOHSA_BOOTSTRAP_PASSWORD` (secreto).
El puerto usa PORT y los enlaces usan RENDER_EXTERNAL_URL.

El servicio gratuito utiliza archivos efímeros: los cambios en pedidos, usuarios y fotografías se pierden al reiniciar o desplegar. No usar como sistema operativo definitivo sin migrar a almacenamiento persistente.
La variable de contraseña de arranque restablece las claves al iniciar. Retirarla después de configurar usuarios únicamente cuando exista almacenamiento persistente.

Los datos cargados llegaron en la raíz. El build los copia a data/; las fotografías históricas adicionales no fueron incluidas en el repositorio.
