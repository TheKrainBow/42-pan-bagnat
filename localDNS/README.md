# Local DNS Helper

Module pages now live on hostnames such as `https://<slug>.modules.localhost`.
Browsers will only resolve that wildcard if your DNS answers for
`*.modules.localhost`. The `localDNS/dnsmasq.conf` file configures a tiny
dnsmasq instance for that purpose.

## Running dnsmasq with Docker

```bash
docker run --rm \
  --name pb-dns \
  -p 127.0.0.1:53:53/udp \
  -v "$(pwd)/localDNS/dnsmasq.conf":/etc/dnsmasq.conf:ro \
  andyshinn/dnsmasq -k
```

What the config does:
- `address=/modules.localhost/127.0.0.1` — responds with `127.0.0.1` for every
  hostname that ends with `.modules.localhost` (including the bare
  `modules.localhost`).
- `listen-address=127.0.0.1` + `bind-interfaces` — keep the resolver local to
  your machine.

After starting the container:
1. Point your OS to `127.0.0.1` for DNS resolution (NetworkManager / `/etc/resolv.conf`
   / macOS network settings).
2. Visit `http://piscine-monitoring.modules.localhost` (or any slug) and it will
   resolve to the Pan Bagnat stack on your own machine.

Stopping the container restores your previous DNS resolver automatically.

> ℹ️ In production we rely on public DNS entries such as
> `*.modules.panbagnat.42nice.fr`. The dnsmasq helper exists only for local
> development where you control both DNS and certificates.
