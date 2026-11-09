> Source code, full documentation and issues: https://github.com/AceRodnel14/subnet-scanner

# subnet-scanner

A small self-hosted web app that pings every host in an IPv4 subnet (`/24` or narrower)
and shows which addresses are online, as a grid with an online/offline count. The last
scan is kept in the browser.

## Quick start
```yaml
services:
  app:
    image: eacer/subnet-scanner:latest
    restart: unless-stopped
    ports: ["8080:8080"]
    env_file: .env
    sysctls:
      net.ipv4.ping_group_range: "0 2147483647"   # unprivileged ICMP for the nonroot user
```
Create `.env` from the variables below (at least `BASE_URL` and `DEFAULT_SUBNET`), then
`docker compose up -d` and open `http://<host>:8080`. Run it on a Linux host: Docker
Desktop's NAT makes LAN scans inaccurate. With `docker run`, pass
`--sysctl net.ipv4.ping_group_range="0 2147483647"`.

## Configuration
| Variable | Default | Description |
| :- | :- | :- |
| `PORT` | `8080` | HTTP port inside the container |
| `BASE_URL` | `http://localhost:8080` | Public URL as users see it |
| `LOG_LEVEL` | `info` | Log level |
| `DEFAULT_SUBNET` | `192.168.1.0/24` | Subnet pre-filled in the UI (IPv4, `/24` or narrower) |
| `ICON` | _(empty)_ | Path to a custom `favicon.ico` inside the container |

Stateless: no volume needed. Health: `/healthz` (liveness), `/readyz` (readiness).

## Tags
`latest` (default, latest stable), `vX.Y`, `vX.Y.Z` (releases). Platforms: `linux/amd64`, `linux/arm64`.
