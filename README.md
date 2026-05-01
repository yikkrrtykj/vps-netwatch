# Komari

Komari is a lightweight, self-hosted server monitoring panel for VPS fleets.
It surfaces live CPU / memory / disk / network / load metrics, monthly traffic
usage, latency and packet-loss probes, and per-server expiration reminders on
a single dashboard.

## Highlights

- Card and table views with live status, network speed, monthly traffic, and
  remaining subscription time.
- Built-in ICMP / TCP / HTTP latency probes with per-task loss and percentile
  tracking.
- Per-server detail page with historical CPU / RAM / load / network charts and
  multi-target ping comparison.

## Docker

The GitHub Actions workflow publishes images to:

```bash
ghcr.io/komari-monitor/komari
```

Run the panel:

```bash
docker run -d \
  --name komari \
  --restart unless-stopped \
  -p 25774:25774 \
  -v ./data:/app/data \
  ghcr.io/komari-monitor/komari:latest
```

## Notes

The working rollback point before the latest cleanup is tagged as
`pre-komari-cleanup-20260502`.
