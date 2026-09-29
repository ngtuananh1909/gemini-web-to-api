<p align="center">
  <img src="assets/gemini.png" width="400" alt="Gemini Logo">
</p>

<h1 align="center">Gemini Web API Gateway</h1>

<p align="center">
  Turn your authenticated Gemini Web session into an OpenAI-compatible local API.
</p>

<p align="center">
  <a href="https://github.com/ngtuananh1909/gemini-web-to-api/releases"><img src="https://img.shields.io/github/v/release/ngtuananh1909/gemini-web-to-api?style=flat-square&logo=github&color=3670ad" alt="Release"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/badge/Go-1.25.1-00ADD8?style=flat-square&logo=go" alt="Go Version"></a>
  <a href="https://www.docker.com/"><img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=flat-square&logo=docker" alt="Docker"></a>
  <a href="https://github.com/ngtuananh1909/gemini-web-to-api/blob/main/LICENSE"><img src="https://img.shields.io/github/license/ngtuananh1909/gemini-web-to-api?style=flat-square&color=orange" alt="License"></a>
</p>

> [!NOTE]
> This unofficial project is not affiliated with Google. It uses browser session cookies and reverse-engineered Gemini Web behavior that may break when Google changes the site. Treat cookies as account credentials and review [Google's Terms of Service](https://policies.google.com/terms).

## What it does

The gateway uses one signed-in Gemini Web session and presents it through these
protocols:

- OpenAI-compatible chat, models, and image-generation endpoints under `/openai/v1`.
- Gemini-native models, content generation, and research endpoints under `/gemini/v1beta`.
- Claude-compatible messages, token counting, and models endpoints under `/claude/v1`.

Model IDs are discovered from the signed-in account. Clients should list models
at startup and use an ID returned by the gateway instead of hard-coding a model
name from an older Gemini API release.

## Quick start with Docker Compose

### 1. Clone and configure

```bash
git clone https://github.com/ngtuananh1909/gemini-web-to-api.git
cd gemini-web-to-api
cp .env.example .env
```

Get a complete `Cookie` request header from a signed-in
[Gemini Web](https://gemini.google.com) tab:

1. Open Developer Tools and select **Network**.
2. Reload Gemini and select a `batchexecute?rpcids=otAQ7b` request.
3. Copy the complete `Cookie` request header into `GEMINI_COOKIES`.
4. If the request URL contains `/u/2/`, set `GEMINI_AUTH_USER=2`. Keep the
   cookie and account slot from the same browser tab.

Keep cookies, `.env`, and the gateway data directory private. The cookies can
authorize requests as the Google account that supplied them.

Leave `API_KEY` empty for the usual local setup. The gateway generates a key,
persists it, and shows it on the local dashboard. To choose your own stable key,
set it in `.env`:

```env
API_KEY=replace-with-a-long-random-value
```

Authentication is enabled by default. `API_KEY` is optional as configuration:
when it is empty, the gateway creates and persists a local key under
`GATEWAY_DATA_DIR`. Set a stable value when scripts or remote clients need a
known key. Requests may send it as `Authorization: Bearer <key>` or
`x-api-key: <key>`. The local dashboard provides the key rotation control for
the generated key.

### 2. Start the gateway

```bash
docker compose up -d
docker compose logs -f gateway
```

Compose binds the host port to `127.0.0.1:4981` and sets `HOST=0.0.0.0`
inside the container. Gateway state is fixed at `/home/appuser/.gateway` in
Compose; it and `.cookies` are persisted in named volumes owned by the non-root
container user. Compose's `DASHBOARD_LOCAL_PORT=true` asserts this loopback-only
host mapping; keep the dashboard disabled if you publish the port publicly.
Earlier `.cookies` bind-mount files are not copied into the new named volume;
the gateway can rebuild its cache from `GEMINI_COOKIES` on first start.

The local dashboard is available at [http://127.0.0.1:4981/](http://127.0.0.1:4981/)
when `DASHBOARD_ENABLED=true`. It is intended for local administration and
should not be published directly to the internet.

### 3. Make a request

Copy the API key from the dashboard, then use a model returned by the models endpoint:

```bash
export API_KEY='YOUR_GATEWAY_API_KEY'
curl -s http://127.0.0.1:4981/openai/v1/models \
  -H "Authorization: Bearer $API_KEY"
```

Then pass one of the returned IDs to the OpenAI-compatible endpoint:

```bash
curl -X POST http://127.0.0.1:4981/openai/v1/chat/completions \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"MODEL_ID_FROM_THE_LIST","messages":[{"role":"user","content":"Hello!"}]}'
```

The interactive API reference is at `/docs`; the raw specification is at
`/openapi.json`.

## Dashboard and remote API access

The dashboard is a local control surface. Keep the default `HOST=127.0.0.1`
when the gateway and its clients run on the same machine. A remote application
should use a separately secured reverse proxy that forwards only API routes.
If the gateway itself must bind to a network interface, set `HOST=0.0.0.0`,
`DASHBOARD_ENABLED=false`, restrict access with a firewall or reverse proxy,
and keep authentication enabled.

To run API-only, set:

```env
DASHBOARD_ENABLED=false
```

This leaves the protocol endpoints available while disabling the dashboard.

## Configuration

`.env.example` lists the complete set of supported settings. The most useful
values are:

| Variable | Default | Purpose |
| --- | --- | --- |
| `HOST` | `127.0.0.1` | Listener address. Compose overrides this inside the container. |
| `PORT` | `4981` | HTTP listener port. |
| `API_AUTH_ENABLED` | `true` | Require an API key on protected routes. |
| `API_KEY` | empty | Optional configured key; an empty value uses a persisted local key. |
| `GATEWAY_DATA_DIR` | `.gateway` | Gateway state path for direct Go runs; Compose uses its fixed named volume path. |
| `DASHBOARD_ENABLED` | `true` on loopback | Serve the local dashboard; defaults off for public listener addresses. |
| `GEMINI_COOKIES` | — | Complete Gemini Web `Cookie` header; required. |
| `GEMINI_AUTH_USER` | empty | Account slot from the Gemini Web URL, such as `2` for `/u/2/app`. |
| `GEMINI_REFRESH_INTERVAL` | `5` | Cookie/session refresh interval in minutes. |
| `GEMINI_MAX_RETRIES` | `3` | Maximum retry attempts for an upstream request. |
| `GEMINI_TEMPORARY` | `false` | Use stateless/incognito mode for requests. |
| `RATE_LIMIT_ENABLED` | `false` | Enable request rate limiting. |
| `RATE_LIMIT_WINDOW_MS` | `60000` | Rate-limit window in milliseconds. |
| `RATE_LIMIT_MAX_REQUESTS` | `10` | Requests allowed in one window. |

Environment variables override `.env` values. Never commit a populated `.env`.

## SDK examples

The files in [`examples/`](examples/) read `API_KEY` from the environment,
discover a current model, and call the matching protocol endpoint:

```bash
export API_KEY='YOUR_GATEWAY_API_KEY'
python examples/openai_client.py
python examples/gemini_client.py
python examples/claude_client.py
python examples/deep_search_client.py
```

Install the client libraries used by the example you want to run. For the
OpenAI-compatible Python client:

```bash
python -m pip install openai
```

For image generation and base64 image input, see
[`docs/image-generation.md`](docs/image-generation.md).

## Build from source

Go 1.25.1 is required:

```bash
go test ./...
go vet ./...
go build -o gemini-web-to-api ./cmd/server/main.go
```

Run directly with a populated `.env`:

```bash
go run ./cmd/server/main.go
```

Live end-to-end tests need valid cookies, network access, and real Gemini
quota:

```bash
E2E=1 go test ./tests/e2e -v -count=1 -timeout 15m
```

## Author and license

Created by **Nguyễn Tuấn Thành** ([@ntthanh2603](https://github.com/ntthanh2603)).
Contributions are welcome through the
[GitHub repository](https://github.com/ngtuananh1909/gemini-web-to-api).

This project is licensed under the [MIT License](LICENSE).
