# Netforge

Learn Go and networking by building a proxy that later becomes a VPN.

## Now

Step 2 (HTTP proxy) done: forward proxy on `127.0.0.1:8080` for `http://` and `https://` (via `CONNECT`). Next: SOCKS5 (step 3).

```bash
go run . proxy                                             # terminal 1
curl.exe -x http://127.0.0.1:8080 http://example.com/      # terminal 2
curl.exe -x http://127.0.0.1:8080 https://example.com/
```

## Roadmap

### 1 — TCP
Echo, then copy both ways (`Listen` / `Dial` / goroutines).

- [x] 1.1 Echo server, one client at a time
- [x] 1.2 One goroutine per client
- [x] 1.3 Split into files, `netforge echo` command
- [x] 1.4 Minimal client: dial, send, read
- [x] 1.5 Byte stream: one `Read` is not one message
- [x] 1.6 Interactive client, both directions at once
- [x] 1.7 Half-close (FIN)
- [x] 1.8 Failures: refused vs timeout, DNS

### 2 — HTTP proxy
Plain HTTP, then `CONNECT`. Requests, headers.

- [x] 2.1 TCP forwarder: accept, dial a fixed target, relay both ways
- [x] 2.2 Read the HTTP request (request line, headers, `\r\n` framing)
- [x] 2.3 Plain HTTP proxy: dial the destination from the request, forward it
- [x] 2.4 `CONNECT`: HTTPS through the proxy

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

Available:

```bash
netforge echo
netforge dial [addr]
netforge proxy
```

Planned:

```bash
netforge socks
netforge vpn client
netforge vpn server
```

One small step at a time. VPN is not a proxy.
