# subnet-scanner

A small self-hosted web app that pings every host in an IPv4 subnet and shows which
addresses are online. Enter a subnet in CIDR notation (`/24` or narrower); the app
pings each host up to 3 times, 50 at a time, and shows the results as a grid of up to
25 columns with an online/offline count. The last scan is kept in your browser, so it
survives a page reload.

## Self-hosting

### Requirements
- Docker with the Compose plugin, on `linux/amd64` or `linux/arm64`, on a **Linux host**.
  Docker Desktop (Windows/macOS) puts containers behind a VM NAT, so scans of your LAN
  are inaccurate there.
- The app pings with unprivileged ICMP sockets as the non-root user (GID 65532). The
  `net.ipv4.ping_group_range` sysctl must include that GID. The Compose file sets it; with
  `docker run` pass `--sysctl net.ipv4.ping_group_range="0 2147483647"`. No extra
  capabilities or root are needed.
- With host networking (`network_mode: host`), Docker rejects `net.*` sysctls: set
  `net.ipv4.ping_group_range` on the host instead.
- Optional: a reverse proxy in front for HTTPS. The app itself only serves plain HTTP
  on port `8080`; set `BASE_URL` to the public URL.

### Quick start
> **Create `.env` from `.env.example` and fill it in before the first start** (at least
> `BASE_URL` and `DEFAULT_SUBNET`; there are no secrets).

```bash
mkdir subnet-scanner && cd subnet-scanner
curl -fsSLO https://raw.githubusercontent.com/AceRodnel14/subnet-scanner/main/docker-compose.yml
curl -fsSL -o .env https://raw.githubusercontent.com/AceRodnel14/subnet-scanner/main/.env.example
# edit .env now: at least BASE_URL and DEFAULT_SUBNET
docker compose pull
docker compose up -d
```
Open `http://<host>:8080`. Check health with `curl -f http://<host>:8080/healthz`.

### Configuration
| Variable | Default | Description |
| :- | :- | :- |
| `PORT` | `8080` | HTTP listen port inside the container |
| `BASE_URL` | `http://localhost:8080` | Public URL users see (through the proxy) |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `DEFAULT_SUBNET` | `192.168.1.0/24` | Subnet pre-filled in the UI; IPv4, `/24` or narrower (checked at startup) |
| `ICON` | _(empty)_ | Path inside the container to a custom `favicon.ico`; empty uses the built-in icon |
| `IMAGE` | `eacer/subnet-scanner:latest` | Image (and tag) Compose runs or builds |
| `APP_BIND` / `APP_PORT` | `0.0.0.0` / `8080` | Host address and port to publish on |

Custom icon: uncomment the volume in `docker-compose.yml`
(`./favicon.ico:/icon/favicon.ico:ro`) and set `ICON=/icon/favicon.ico`. The file must be
readable by UID 65532.

### Scanning
- IPv4 only, `/24` or narrower (at most 254 hosts). Wider subnets and IPv6 are rejected.
- A host that doesn't answer takes about 3 s (3 probes, 1 s each), so a mostly empty
  `/24` can take 15-20 s.
- If the ping sysctl isn't set, a scan fails with a message naming
  `net.ipv4.ping_group_range` instead of showing every host offline.

### Kubernetes
> **Required when deployed in Kubernetes:** set `net.ipv4.ping_group_range` in the pod
> spec. Without it every scan fails with `503` (ICMP ping sockets not permitted).

It is a "safe" sysctl since Kubernetes 1.18, so no kubelet flag is needed; containerd and
CRI-O often default it to disabled, so set it explicitly:
```yaml
spec:
  securityContext:
    sysctls:
      - name: net.ipv4.ping_group_range
        value: "0 2147483647"
```
It can't be combined with `hostNetwork`. A NetworkPolicy must allow ICMP egress, and a
pod only reaches the subnets its network can route to.

### Data and backups
There is no server-side data: the last scan lives in the browser's `localStorage`.
Nothing to back up.

### Upgrading
`latest` is the latest stable image and what Compose uses by default. Release tags
`vX.Y` and `vX.Y.Z` are pushed when a release is tagged: pin `IMAGE` to `vX.Y` for patch
updates only, then `docker compose pull && docker compose up -d`.

### Health endpoints
- `/healthz`: the process is up (liveness).
- `/readyz`: the app is ready (readiness).

### API
- `GET /api/info`: app info, including `default_subnet` and `has_icon`.
- `POST /api/scan` with `{"subnet":"192.0.2.0/28"}`: returns `[{"ip":"…","alive":true}]`,
  sorted by IP. `400` for an invalid or too-wide subnet, `503` when ICMP isn't permitted.

## Development
```bash
cp .env.example .env
docker compose up --build   # the whole app, as self-hosters run it
go test ./...
go run ./cmd/app            # backend on :8080
./manual_cicd_lint.sh       # the CI lint/test jobs, locally in throwaway containers
```
With the web UI (`web/`): `npm --prefix web install`, then `npm --prefix web run dev`
(UI on :5173, API proxied to :8080) next to `go run ./cmd/app`; `npm --prefix web test`.
`go run`/`go test` need `web/dist`: run `npm --prefix web run build` once, or create
an empty `web/dist/.keep`.
CI (gitlab-actions `templates/app.yml`): lint and tests on branches and merge requests;
on the default branch and release tags a test build with a smoke run, the multi-arch
image push to Docker Hub and the GitHub sync.

---
_Developed with assistance from Claude (Claude Opus 5.5)._
