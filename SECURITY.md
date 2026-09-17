# Security Policy

## Supported Versions

Only the latest minor line receives security fixes. Older versions are
supported best-effort while an upgrade path exists; the module is 0.x, so
bumping across minors is expected to be cheap (see the README "Upgrading"
section for the rare behavior changes).

| Version | Supported        |
| ------- | ---------------- |
| 0.9.x   | Yes              |
| < 0.9   | No — please bump |

## Reporting a Vulnerability

Email **git@lars.software** (the maintainer identity on every commit) with
`[security]` in the subject, or open a private GitHub security advisory at
<https://github.com/LarsArtmann/go-health-dashboard/security/advisories/new>
if private vulnerability reporting is enabled on the repository.

Please include: a description, reproduction steps or a proof of concept,
affected versions, and your assessment of severity and impact.

You will get an acknowledgement within 3 business days. Findings are
coordinated-disclosed: a fix and a CHANGELOG entry land first, and details
are published within 90 days of the report.

## Scope

The dashboard is a display layer over your health data. Reports involving
the served HTML and SSE patch path (XSS/CSP), the JSON/export endpoints,
the webhook sink, or the metrics exposition are especially welcome. Please
do not test against deployments you do not own.
