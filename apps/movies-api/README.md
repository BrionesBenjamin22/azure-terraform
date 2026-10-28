# Movies API

Servicio en Go para consultar la API de TMDB y exponer información de películas.

## Estado

Implementación incremental en curso. Actualmente dispone de configuración validada, health checks, cierre ordenado e integración con TMDB para películas populares.

## Ejecución local

Requisitos:

- Go 1.27 o una versión compatible con la declarada en `go.mod`.

Desde esta carpeta:

```powershell
go run ./cmd/api
```

El servidor escucha por defecto en `:8080`. Los valores pueden modificarse mediante las variables documentadas en `.env.example`.

## Configuración

| Variable | Valor predeterminado | Regla |
| --- | --- | --- |
| `HTTP_PORT` | `8080` | Entero entre 1 y 65535. |
| `HTTP_READ_TIMEOUT_SECONDS` | `5` | Entero positivo. |
| `HTTP_WRITE_TIMEOUT_SECONDS` | `10` | Entero positivo. |
| `HTTP_IDLE_TIMEOUT_SECONDS` | `60` | Entero positivo. |
| `SHUTDOWN_TIMEOUT_SECONDS` | `10` | Entero positivo. |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` o `error`. |
| `TMDB_BASE_URL` | `https://api.themoviedb.org/3` | URL HTTP(S) absoluta, sin credenciales, query ni fragmento. |
| `TMDB_API_TOKEN` | Sin valor | Token obligatorio; nunca se registra. |
| `TMDB_DEFAULT_LANGUAGE` | `es-AR` | Código de idioma o idioma-país. |
| `TMDB_TIMEOUT_SECONDS` | `5` | Entero positivo. |

El proceso valida toda la configuración antes de aceptar tráfico. `.env.example` es documentación y no se carga automáticamente.

## Contrato disponible

### `GET /health/live`

Confirma únicamente que el proceso HTTP está respondiendo. No consulta configuración, red ni TMDB.

Respuesta exitosa, `200 OK`:

```json
{
  "status": "ok"
}
```

Los métodos no admitidos reciben `405 Method Not Allowed` mediante `http.ServeMux`.

### `GET /health/ready`

Confirma que la configuración y las dependencias internas se inicializaron correctamente. No consulta TMDB ni depende de servicios externos.

Respuesta exitosa, `200 OK`:

```json
{
  "status": "ready"
}
```

### `GET /api/v1/movies/popular?page=1`

Obtiene una página de películas populares de TMDB y devuelve como máximo sus primeros 9 resultados. `page` es opcional y su valor predeterminado es `1`; debe aparecer una sola vez y ser un entero mayor o igual que 1.

```json
{
  "data": [],
  "meta": {
    "page": 1,
    "returned": 0,
    "has_next": false
  }
}
```

`page` representa directamente la página solicitada a TMDB. La API no combina páginas ni ofrece `page_size`.

## Contrato HTTP transversal

Cada respuesta incluye `X-Request-ID`. El servicio conserva identificadores enviados por el cliente únicamente cuando contienen entre 1 y 64 caracteres alfanuméricos, `.`, `_` o `-`; en cualquier otro caso genera uno nuevo.

Los errores utilizan el siguiente envelope:

```json
{
  "error": {
    "code": "NOT_FOUND",
    "message": "Lo sentimos. No pudimos encontrar el recurso solicitado. Verifique la dirección e intente nuevamente.",
    "request_id": "identificador-correlacionable"
  }
}
```

Cada request genera un log estructurado con `request_id`, `method`, `path`, `status` y `duration_ms`. No se registran query strings, headers, cuerpos ni secretos.

## Diseño actual

- Punto de entrada: `cmd/api/main.go`.
- Router: `http.ServeMux` de la biblioteca estándar.
- Configuración tipada y validada en `internal/config`.
- Health checks en `internal/httpapi`.
- Modelos neutrales de películas en `internal/movies`.
- Adaptador externo encapsulado en `internal/tmdb`.
- Timeouts explícitos de lectura de encabezados, lectura, escritura e inactividad.
- Logging estructurado JSON mediante `log/slog`, con middleware HTTP y correlación por request.
- Lifecycle: Kubernetes envía `SIGTERM`, la aplicación deja de aceptar tráfico y ejecuta `http.Server.Shutdown` con un plazo limitado; si el plazo vence, fuerza el cierre.
- Pruebas de handlers, configuración, timeouts y cierre ordenado.
- Sin dependencias externas.

No se agrega un middleware propio de recuperación de panic en esta etapa. `net/http` ya aísla panics por conexión y un panic posterior al inicio de la respuesta no permite garantizar un envelope JSON válido. La decisión se revisará si aparece una necesidad concreta de recuperación controlada.

## Adaptador TMDB

`internal/tmdb` implementa un cliente concreto y reutilizable con `http.Client`, contexto por request y timeout global. Los DTO de TMDB son privados; el resto de la aplicación recibe modelos neutrales de `internal/movies`.

Contrato oficial verificado:

- autenticación de aplicación mediante [Bearer token](https://developer.themoviedb.org/docs/authentication-application);
- [películas populares](https://developer.themoviedb.org/reference/movie-popular-list) mediante `GET /movie/popular`;
- [búsqueda de películas](https://developer.themoviedb.org/reference/search-movie) mediante `GET /search/movie` con `query`, `page`, `language` e `include_adult=false`;
- [detalle de película](https://developer.themoviedb.org/reference/movie-details) mediante `GET /movie/{movie_id}`;
- clasificación basada en los [errores oficiales de TMDB](https://developer.themoviedb.org/docs/errors).

El adaptador distingue recurso inexistente, autenticación rechazada, límite de cuota, timeout, indisponibilidad y respuesta inválida. Los logs del proveedor contienen solamente `provider`, `operation`, `status` y `duration_ms`; no incluyen URL, query, token ni cuerpo.

## Validaciones y errores actuales

| Caso | Estado | Código público |
| --- | --- | --- |
| Página inválida | `400` | `INVALID_PAGE` |
| Credenciales del proveedor rechazadas | `502` | `PROVIDER_AUTHENTICATION_ERROR` |
| Límite del proveedor alcanzado | `503` | `PROVIDER_RATE_LIMITED` |
| Timeout del proveedor | `504` | `PROVIDER_TIMEOUT` |
| Proveedor no disponible | `503` | `PROVIDER_UNAVAILABLE` |
| Respuesta externa inválida | `502` | `PROVIDER_INVALID_RESPONSE` |

Las entradas inválidas se rechazan antes de llamar a TMDB. Todos los errores conservan el envelope y el `request_id` del contrato transversal.

## Validación

```powershell
gofmt -w .
go test ./...
go vet ./...
```

## Alcance técnico previsto

- Servicio sin estado.
- Configuración mediante variables de entorno.
- Réplicas y balanceo de carga en Kubernetes.
- Escalamiento horizontal.

El contrato HTTP de películas, sus validaciones y sus errores se incorporarán en las siguientes tareas.
