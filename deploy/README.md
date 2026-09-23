# Deploy — Prometheus + Grafana demo stack

A three-container demo: the example dashboard, Prometheus scraping its
`/health/metrics` endpoint every 5s, and Grafana with a fully provisioned
8-panel dashboard over the scraped series. It exists to prove the metrics
contract end to end — the panels execute against a live scrape, not
mocked data (verified 2026-09-23; see `docs/screenshot-grafana.png`).

## Run it

```bash
docker compose -f deploy/docker-compose.yml up --build
```

| Service    | URL                            | Notes                                   |
| ---------- | ------------------------------ | --------------------------------------- |
| Dashboard  | http://localhost:8080/health   | demo probe with pass/warn/fail checks   |
| Metrics    | http://localhost:8080/health/metrics | what Prometheus scrapes           |
| Prometheus | http://localhost:9090          | target `go-health-dashboard`, 5s scrape |
| Grafana    | http://localhost:3000          | anonymous **Viewer**, no login          |

Host ports are hard-coded in the compose file. If any of 8080/9090/3000
is taken on your machine, remap without editing the repo:

```bash
cat > /tmp/override.yml <<'EOF'
services:
  dashboard:
    ports: !override ["18080:8080"]
  prometheus:
    ports: !override ["19090:9090"]
  grafana:
    ports: !override ["13000:3000"]
EOF
docker compose -f deploy/docker-compose.yml -f /tmp/override.yml up --build
```

(The `!override` tag replaces the port list; without it compose MERGES
both lists and still tries to bind the taken port.)

## Layout

```
deploy/
├── docker-compose.yml            # three services, digest-pinned images
├── prometheus.yml                # 5s scrape of dashboard:8080/health/metrics
└── grafana/provisioning/
    ├── datasources/prometheus.yml     # Prometheus datasource, uid "prometheus"
    ├── dashboards/provider.yml        # dashboard provider (side-loaded JSON)
    ├── dashboards/go-health-dashboard.json  # the 8-panel dashboard
    └── alerting/
        ├── placeholder.yaml           # keeps the dir non-empty for Grafana
        └── dashboard-down-alert.yaml.example  # inert threshold-alert example
```

## The dashboard panels

All panels query the `prometheus` datasource; the metric contract is
documented in `docs/metrics.md`.

1. **Overall health** — stat: `dashboard_health_up` (1 = all passing)
2. **Status** — stat: `dashboard_health_status` (2 pass / 1 warn / 0 fail)
3. **Passing checks** — stat: sum of passing `dashboard_health_check`
4. **Last batch wall-clock** — stat: `dashboard_health_latency_ms`
5. **Shutdown flag** — stat: `dashboard_health_shutting_down`
6. **Last duration per check** — timeseries: `dashboard_health_check_last_duration_seconds` per `check`
7. **Check duration percentiles** — timeseries: p50/p95 from the `dashboard_health_check_duration_seconds` histogram
8. **Check executions per second** — timeseries: `rate(..._count[1m])`

## Security posture

- **Grafana runs anonymous with the Viewer role** (`GF_AUTH_ANONYMOUS_*`)
  and basic auth disabled: anyone who can reach port 3000 can browse the
  demo dashboard but cannot edit panels, datasources, or alerts. This is
  a demo convenience — do not port-forward it to the internet. Real
  deployments should front Grafana with auth (oauth proxy, basic auth,
  or Grafana's own users).
- The demo dashboard itself is intentionally UNauthenticated (no
  `DEMO_AUTH`), so Prometheus can scrape it without a bearer token.
- No volumes persist state: `docker compose down` (add `-v` anyway for
  hygiene) erases the Grafana database and both data dirs. Nothing from
  the host is mounted read-write.

## Dependency bumps

Both images are **digest-pinned** (`repo:tag@sha256:...`), and
`.github/dependabot.yml` carries `docker` ecosystem entries for `/` and
`/deploy`, so image and base-image bumps arrive as proposed PRs, never
silent pulls. To bump by hand, resolve the manifest-LIST digest (not a
single-platform digest):

```bash
docker buildx imagetools inspect <image:tag> | grep '^Digest:'
```

## Teardown

```bash
docker compose -f deploy/docker-compose.yml down -v
```

## Threshold alert example

`grafana/provisioning/alerting/dashboard-down-alert.yaml.example` ships
a worked, live-verified rule: it fires when
`min(dashboard_health_up) < 1` for one minute. Grafana only reads
`.yaml`/`.yml` from the alerting directory, so the `.yaml.example` file
stays inert. Activate it with:

```bash
cp deploy/grafana/provisioning/alerting/dashboard-down-alert.yaml.example \
   deploy/grafana/provisioning/alerting/dashboard-down-alert.yaml
docker compose -f deploy/docker-compose.yml restart grafana
```

Note the schema quirk that cost a live-debugging round: provisioned
alert rules take `relativeTimeRange` as **integer seconds** (`from: 300`),
not duration strings (`"5m"` 422s the whole provisioning pass). The
demo's flapping redis check makes `dashboard_health_up` 0, so the
activated rule goes Pending immediately — that is the alert working.

## Boot verification history

- 2026-09-23: first full e2e boot. Found and fixed the stale
  `golang:1.26` base (go.mod's 1.27.1 floor broke `go mod download`),
  added the readiness healthchecks, digest pins, and the alert example
  (activated, evaluated `Pending` against the live failing probe, then
  deactivated). Screenshot: `docs/screenshot-grafana.png`.
