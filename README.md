# Netforge

Learn Go and networking by building a proxy that later becomes a VPN.

## Now

TCP forwarder on `127.0.0.1:8080`, relaying every client to a fixed target (the echo server by default). Next: read the HTTP request (step 2.2).

```bash
go run . echo                  # terminal 1: echo server on 127.0.0.1:9000
go run . proxy                 # terminal 2: proxy on 127.0.0.1:8080
go run . dial 127.0.0.1:8080   # terminal 3: dial -> proxy -> echo
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
- [ ] 2.2 Read the HTTP request (request line, headers, `\r\n` framing)
- [ ] 2.3 Plain HTTP proxy: dial the `Host`, forward the request
- [ ] 2.4 `CONNECT`: HTTPS through the proxy

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
netforge proxy [target]
```

Planned:

```bash
netforge socks
netforge vpn client
netforge vpn server
```

One small step at a time. VPN is not a proxy.
