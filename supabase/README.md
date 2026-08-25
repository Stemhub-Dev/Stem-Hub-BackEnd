# Auth local (Supabase CLI)

Reemplaza a Supabase Cloud para desarrollo local: `supabase start` levanta
el mismo stack de Auth que corre en la nube (GoTrue + Kong + Postgres
propio), para que `SUPABASE_URL` apunte a `http://127.0.0.1:54321` en vez
de `https://TU_PROJECT_REF.supabase.co`.

`internal/middleware/auth_middleware.go` valida los tokens exclusivamente
en `ES256` y exige que el claim `iss` coincida con `SUPABASE_URL`. La CLI
firma en `ES256` e incluye `iss` por defecto — no requiere ningún ajuste de
configuración de Auth en sí. Lo que sí requirió un ajuste puntual (ver
abajo) es cómo el Backend, corriendo dentro de un contenedor Docker, llega
a validar ese `iss`.

## Dos Postgres, no uno

La CLI **no puede apuntar a un Postgres externo**: siempre crea y gestiona
el suyo propio (puerto `54322`), separado del Postgres de negocio que
levanta el `docker compose` del Backend (puerto `5432`). No hay foreign
key real entre `usuario.idautenticacion` (tabla de negocio) y
`auth.users.id` (Postgres de Supabase) — son bases distintas conectadas
solo por el valor del UUID, igual que en Supabase Cloud.

Esto implica **dos comandos** para levantar el entorno completo:

```bash
npx supabase start        # Auth: Postgres propio, GoTrue, Kong, Studio...
docker compose up -d      # Postgres de negocio + Backend
```

## `SUPABASE_URL` debe usar `127.0.0.1`, no `localhost`

La Supabase CLI graba el `iss` del token como `http://127.0.0.1:54321/auth/v1`
— es un valor fijo, no configurable (limitación conocida de la CLI:
[supabase/cli#3118](https://github.com/supabase/cli/issues/3118),
[#4006](https://github.com/supabase/cli/issues/4006)). Aunque
`localhost` y `127.0.0.1` apunten al mismo lugar, son *strings* distintos
y el middleware compara el `iss` como texto exacto. Por eso
`SUPABASE_URL` tiene que ser literalmente `http://127.0.0.1:54321`.

## Por qué existe `SUPABASE_JWKS_BASE_URL`

Con el Backend corriendo en un contenedor Docker, `127.0.0.1` dentro de
ese contenedor apunta al propio contenedor, no al host — así que no puede
usarse para *descargar* el JWKS, aunque sí sea el valor correcto para
*validar* el `iss`. Y ninguna redirección de red (`extra_hosts`, DNS,
`network_mode: host` en Docker Desktop) permite hacer que `127.0.0.1`
resuelva a otra cosa: es tratamiento especial del sistema operativo del
contenedor, no config de Docker.

La solución: separar de dónde se **descarga** el JWKS de qué **issuer** se
**valida**. `auth_middleware.go` acepta un parámetro opcional
(`SUPABASE_JWKS_BASE_URL`) para el primero; el segundo sigue derivándose
siempre de `SUPABASE_URL`. El `compose.yaml` conecta el contenedor
`backend` a la red Docker que crea `supabase start`
(`supabase_network_Stem-Hub-BackEnd`) y usa el nombre de servicio `kong`
para bajar el JWKS por ahí, mientras sigue validando `iss` contra
`127.0.0.1:54321` (el valor real del token).

Con Supabase Cloud esta variable se deja vacía: JWKS e issuer vuelven a
ser la misma URL, como siempre.

## Setup inicial

Requiere Docker (ya usado por el resto del proyecto). No hace falta
instalar la CLI globalmente, `npx` la descarga on-demand:

```bash
npx supabase start
```

La primera vez descarga varias imágenes y tarda unos minutos. Al terminar
imprime las credenciales del entorno local (`API_URL`, `ANON_KEY`, etc.) —
no hace falta copiar nada a mano: `.env.example` ya trae los valores
correctos (`SUPABASE_URL`, `SUPABASE_JWKS_BASE_URL`).

## Servicios expuestos

| Servicio | URL local |
|---|---|
| API del Backend | http://localhost:8080 |
| Auth (vía Kong) | http://127.0.0.1:54321/auth/v1 |
| Supabase Studio (panel admin) | http://127.0.0.1:54323 |
| Postgres de Auth | postgresql://postgres:postgres@127.0.0.1:54322/postgres |
| Bandeja de mails de prueba | http://127.0.0.1:54324 |

En `supabase/config.toml` están deshabilitados los servicios que no usa
este proyecto (`storage`, `realtime`, `edge_runtime`, `analytics`) para
levantar menos contenedores.

## Crear un usuario de prueba

```bash
curl -X POST http://127.0.0.1:54321/auth/v1/signup \
  -H "Content-Type: application/json" \
  -H "apikey: $(npx supabase status -o env | grep ANON_KEY | cut -d= -f2- | tr -d '\"')" \
  -d '{"email":"dev@stemhub.local","password":"Password123!"}'
```

Con `enable_confirmations = false` (ya configurado en `config.toml`) el
usuario queda confirmado sin pasar por el mailer. La respuesta incluye
`access_token`: usarlo como `Authorization: Bearer <token>` contra la API
del Backend. También se puede crear un usuario a mano desde **Studio**
(http://127.0.0.1:54323 → Authentication → Add user).

## Comandos útiles

```bash
npx supabase status   # ver credenciales y URLs sin volver a levantar nada
npx supabase stop     # apagar el stack (agrega --no-backup para no persistir datos)
```

## Volver a Supabase Cloud

Alcanza con cambiar en `.env`:

```env
SUPABASE_URL=https://TU_PROJECT_REF.supabase.co
SUPABASE_JWKS_BASE_URL=
```

y no correr `supabase start`. El middleware no requiere ningún otro
cambio.
