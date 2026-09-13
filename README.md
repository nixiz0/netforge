# Netforge

Learn Go and networking by building a proxy that later becomes a VPN.

## Now

TCP echo on `:9000`. Then HTTP forward proxy on `:8080`.

## Roadmap

### 1 — TCP
Echo, then copy both ways (`Listen` / `Dial` / goroutines).

### 2 — HTTP proxy
Plain HTTP, then `CONNECT`. Requests, headers, L7.

### 3 — SOCKS5
Binary protocol, IPv4/IPv6, DNS.

### 4 — UDP
Datagrams.

### 5 — Tunnel
Client/server encapsulation.

### 6 — TUN
Raw IP, routing, virtual NIC.

### 7 — VPN
Route IP through the tunnel (MTU, routes).

### 8 — Encryption
AEAD, auth, key exchange.

### 9 — Multi-peer
Peers and routes.

## CLI

```bash
netlab proxy
netlab socks
netlab vpn client
netlab vpn server
```

One small step at a time. VPN is not a proxy.
