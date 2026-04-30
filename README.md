# vps-netwatch

vps-netwatch is a VPS monitoring panel adjusted for route
latency, traffic, remaining time, and lightweight server status checks.

## Current Direction

- Card and table views focus on VPS status, live network speed, traffic quota,
  remaining time, IPv4/IPv6 labels, and latency summaries.
- The target wizard can create ICMP checks from `IP` / domain input and TCP
  checks from `IP:port`.
- The mihomo / Clash helper can read current connections and add discovered
  game server targets to monitoring.

## Docker

The GitHub Actions workflow publishes images to:

```bash
ghcr.io/yikkrrtykj/vps-netwatch
```

Run the panel:

```bash
docker run -d \
  --name vps-netwatch \
  --restart unless-stopped \
  -p 25774:25774 \
  -v ./data:/app/data \
  ghcr.io/yikkrrtykj/vps-netwatch:latest
```

## Notes

The working rollback point before this migration is tagged as
`checkpoint-working-20260430`.
