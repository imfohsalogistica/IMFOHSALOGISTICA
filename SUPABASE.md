# Almacenamiento de Logística IMFOHSA

El servidor mantiene el modelo JSON actual y persiste el estado, los usuarios
(con sus hashes de contraseña) y las fotografías en PostgreSQL. El esquema
`imfohsa_private` está fuera de la Data API, con RLS habilitado y sin permisos
para `anon` ni `authenticated`. La ausencia de políticas públicas es intencional.
La aplicación conserva su autenticación y permisos actuales.

## Activar en Render

En Environment, configure la variable privada `DATABASE_URL` con la cadena de
Session pooler de su proyecto y la contraseña real de la base de datos:

```text
postgresql://postgres.pzeyfhgjojwskriwbeam:[YOUR-PASSWORD]@aws-0-us-east-1.pooler.supabase.com:5432/postgres?sslmode=require
```

Codifique los caracteres especiales de la contraseña para una URL. No guarde la
cadena con contraseña en GitHub. Configure `DATABASE_REQUIRED=1` al activar la
conexión para impedir que el servicio arranque en modo local por una variable
faltante. Luego guarde y despliegue.

La migración `sql/imfohsa_supabase.sql` ya fue aplicada al proyecto
`pzeyfhgjojwskriwbeam` el 30 de septiembre de 2026. Cada tabla vacía se inicializa
una sola vez desde los archivos disponibles en el despliegue; el estado existente
en Supabase nunca se reemplaza con datos iniciales en un reinicio. Antes del
primer despliegue con base de datos, exporte cualquier cambio reciente que solo
exista en el servicio local/Render anterior: los archivos del ZIP no contienen
necesariamente esos cambios. Los adjuntos locales disponibles se importan al
arrancar. `IMFOHSA_BOOTSTRAP_PASSWORD` solo se aplica al crear usuarios inicialmente
en la base de datos, conservando sus cambios de contraseña posteriores.

El sistema mantiene el estado en memoria: use una instancia de Render. Una
comprobación de versión evita que otra instancia sobrescriba cambios durante
un despliegue; ante conflicto se recarga el estado y se solicita reintentar.
Las sesiones siguen en memoria y requieren ingresar nuevamente tras un reinicio.
Las fotografías se guardan como bytea en PostgreSQL y consumen espacio de la base.

## Comprobar

`/api/health` devuelve `storage: "postgresql"` cuando está conectado. Una caída
de la conexión produce HTTP 503. Un fallo de guardado no devuelve éxito ni
continúa guardando en disco. `documents` tiene las claves `state` y `admins`;
`uploads` contiene los adjuntos. Estas tablas están en el esquema privado.

```sh
go test -race ./...
go vet ./...
go build -o server_imfohsa server_imfohsa.go
```

Sin `DATABASE_URL`, el piloto de Windows sigue usando los archivos locales.
