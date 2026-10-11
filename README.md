# Netforge

Learn networking fundamentals (theory) and build a proxy and a VPN in Go (practice).

Docs: [ARCHITECTURE.md](ARCHITECTURE.md) (stack, internals, conventions) · [docs/blueprint.html](docs/blueprint.html) (interactive visual map, open in a browser) · [_networking-lessons/](_networking-lessons/) (theory)

## Theory ↔ practice

Each roadmap stage has its lesson. In the code, `// Theory:` comments point to the exact section a line puts into practice.

| Lesson | Practiced in |
|---|---|
| [1 · Network fundamentals](_networking-lessons/1-network-fundamentals.md) | loopback, DNS, NAT, MTU: all stages |
| [2 · Transport networking](_networking-lessons/2-transport-networking.md) | Stage 1 TCP, Stage 4 UDP |
| [3 · HTTP, TLS and proxies](_networking-lessons/3-http-tls-proxies.md) | Stage 2 HTTP proxy, Stage 3 SOCKS5 |
| [4 · Advanced network concepts](_networking-lessons/4-advanced-network-concepts.md) | Stage 7 VPN (NAT, firewall), Stage 9 multi-peer (NAT traversal) |
| [5 · Tunneling](_networking-lessons/5-tunneling.md) | Stage 5 Tunnel |
| [6 · TUN / TAP](_networking-lessons/6-tun-tap.md) | Stage 6 TUN |
| [7 · VPN architecture](_networking-lessons/7-vpn-architecture.md) | Stage 7 VPN |
| [8 · Security and cryptography](_networking-lessons/8-security-cryptography.md) | Stage 8 Encryption |
| [9 · Multi-peer networking](_networking-lessons/9-multi-peer-networking.md) | Stage 9 Multi-peer |

## Now

All 9 stages done. TCP echo server and client; HTTP/HTTPS forward proxy on `127.0.0.1:8080`; SOCKS5 proxy on `127.0.0.1:1080`; UDP echo server and a client that reports lost replies; a point-to-point UDP tunnel (`127.0.0.1:5300/udp` → `127.0.0.1:7000/udp`, 1468-byte MTU, keepalives); a TUN interface that decodes IPv4 and answers ping; and a VPN between Linux network namespaces (WSL2): routing, NAT and full tunnel, DNS inside the tunnel (`netforge dns`), an X25519 handshake that derives fresh AES-256-GCM session keys, replay protection, and a server with a peer table for several clients, each allowed to send only from its own VPN address.

```bash
go run ./cmd/netforge proxy                                # HTTP proxy
curl.exe -x http://127.0.0.1:8080 https://example.com/

go run ./cmd/netforge socks                                # SOCKS5
curl.exe --socks5-hostname 127.0.0.1:1080 https://example.com/   # the proxy resolves DNS
curl.exe --socks5 127.0.0.1:1080 https://example.com/            # curl resolves DNS itself

go run ./cmd/netforge udpecho                              # UDP echo on 127.0.0.1:9000/udp
go run ./cmd/netforge udpdial                              # each line = one datagram
go run ./cmd/netforge udpdial 10.255.255.1:9000            # nobody answers: silent loss
# in udpdial, type +1472, +65507, +65508 to probe datagram size limits

go run ./cmd/netforge tunnel server                        # far end: 127.0.0.1:7000/udp → udpecho
go run ./cmd/netforge tunnel client                        # near end: entrance 127.0.0.1:5300/udp
go run ./cmd/netforge udpdial 127.0.0.1:5300               # talk to udpecho THROUGH the tunnel
```

TUN (Linux only, inside WSL2):

```bash
git config --global --add safe.directory "$(pwd)"   # once: lets go build read git info on /mnt/c
CGO_ENABLED=0 go build -o /tmp/netforge ./cmd/netforge && sudo /tmp/netforge tun   # terminal 1 (pure Go, static binary)
sudo ip addr add 10.10.0.1/24 dev nf0 && sudo ip link set nf0 up mtu 1468   # terminal 2
ping -c 3 10.10.0.2                                                     # answered by netforge itself
curl -m 2 http://10.10.0.2/                                             # a TCP SYN shows up too
```

The VPN between two network namespaces (Linux, inside WSL2):

```bash
/tmp/netforge genkey > /tmp/server.key && /tmp/netforge pubkey /tmp/server.key > /tmp/server.pub   # server key pair
/tmp/netforge genkey > /tmp/client.key && /tmp/netforge pubkey /tmp/client.key > /tmp/client.pub   # client key pair
sudo bash scripts/lab.sh up                                                              # namespace nfpeer + veth cable
sudo ip netns exec nfpeer /tmp/netforge vpn server /tmp/server.key /tmp/client.pub=10.10.0.1   # terminal 1: server, learns its client
sudo /tmp/netforge vpn client 192.168.100.2:7000 /tmp/client.key /tmp/server.pub     # terminal 2: client, keepalives
sudo bash scripts/lab.sh tun                                                             # client nf0 10.10.0.1, server nf0 10.10.0.2
ping -c 3 10.10.0.2                                                                      # answered by the server's kernel
sudo tcpdump -X -c 2 -ni veth0 udp port 7000                                             # during a ping: only ciphertext on the wire
sudo python3 scripts/replay.py                                                           # replay attack: the copy is dropped
sudo bash scripts/lab.sh route && sudo bash scripts/lab.sh forward                       # server becomes a router
ping -c 3 203.0.113.10                                                                   # a host BEHIND the server
sudo bash scripts/lab.sh nat                                                             # server NATs the VPN subnet
ping -c 3 203.0.113.10                                                                   # the host now sees 203.0.113.1
sudo bash scripts/lab.sh full                                                            # ALL traffic into the tunnel (WSL loses internet)
sudo bash scripts/lab.sh dns                                                             # LEAK: WSL's resolver is not behind nf0
sudo bash scripts/lab.sh full-off                                                        # back to normal
sudo ip netns exec nfpeer /tmp/netforge dns :53                                          # terminal 3: DNS server on the VPN server
dig @10.10.0.2 host.lab                                                                  # inner address: through the tunnel (apt install dnsutils)
dig @192.168.100.2 host.lab                                                              # outer address: same server, beside the tunnel
/tmp/netforge genkey > /tmp/client2.key && /tmp/netforge pubkey /tmp/client2.key > /tmp/client2.pub   # step 9: a second client
sudo bash scripts/lab.sh up2                                                             # its namespace nfc2 + a cable to the server
sudo ip netns exec nfpeer /tmp/netforge vpn server /tmp/server.key /tmp/client.pub=10.10.0.1 /tmp/client2.pub=10.10.0.3   # terminal 1, restarted: 2 peers
sudo ip netns exec nfc2 /tmp/netforge vpn client 192.168.101.2:7000 /tmp/client2.key /tmp/server.pub                    # terminal 4: client 2
sudo bash scripts/lab.sh tun && sudo bash scripts/lab.sh tun2 && sudo bash scripts/lab.sh forward                       # addresses again, server forwards
ping -c 3 10.10.0.3                                                                      # client 1 -> server -> client 2
sudo bash scripts/lab.sh spoof                                                           # client 2 claims 10.10.0.1: dropped
sudo bash scripts/lab.sh down                                                            # cleanup
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

- [x] 3.1 Greeting: method negotiation, binary framing with `io.ReadFull`
- [x] 3.2 Read the CONNECT request (address type, big-endian port)
- [x] 3.3 Share the relay: move it to `internal/relay`
- [x] 3.4 Connect, reply, relay: a working SOCKS5 proxy
- [x] 3.5 DNS: client-side vs proxy-side resolution
- [x] 3.6 Explicit DNS lookups in both proxies, logged with timing

### 4 — UDP
Datagrams.

- [x] 4.1 UDP echo server: one socket, no handshake, sender address per datagram
- [x] 4.2 UDP client: one write = one datagram, boundaries kept
- [x] 4.3 Loss is silent: read deadlines, closed ports
- [x] 4.4 Datagram size: limits and MTU (`+N` probes)

### 5 — Tunnel
Client/server encapsulation.

- [x] 5.1 Tunnel packet format: 4-byte header + payload, encode/decode, first tests
- [x] 5.2 Tunnel server: decapsulate, forward to the inner target, wrap replies
- [x] 5.3 Tunnel client: local entrance, end to end udpdial → tunnel → udpecho
- [x] 5.4 Overhead and tunnel MTU (1468 = 1500 − 32)
- [x] 5.5 Keepalives and tunnel state (5 s keepalive, 15 s peer timeout)

### 6 — TUN
Raw IP, routing, virtual NIC.

- [x] 6.1 Open a TUN interface (`nf0`) and log the raw IP packets routed into it
- [x] 6.2 Decode the IPv4 header (tested parser)
- [x] 6.3 Write path: answer ping from userspace (ICMP echo reply + checksum)
- [x] 6.4 TUN ⇄ tunnel: IP packets through the tunnel between two network namespaces

### 7 — VPN
Route IP through the tunnel (MTU, routes).

- [x] 7.1 `vpn server` / `vpn client`: the server learns its client, keepalives, roaming
- [x] 7.2 The server as a router: IP forwarding, a machine behind the server, return route
- [x] 7.3 NAT on the server
- [x] 7.4 Full tunnel without routing the tunnel into itself
- [x] 7.5 See the DNS leak: the resolver's route bypasses the full tunnel
- [x] 7.6 A DNS server inside the VPN (`netforge dns`, reached at the server's inner address)

### 8 — Encryption
AEAD, replay protection, key exchange.

- [x] 8.1 Seal every VPN packet with AEAD (AES-256-GCM, shared key): confidentiality, integrity, no more hijack
- [x] 8.2 Replay protection: a sliding window over packet counters
- [x] 8.3 Key exchange: X25519 key pairs, a handshake, fresh session keys (forward secrecy)

### 9 — Multi-peer
Peers and routes.

- [x] 9.1 Peer table: one server, two clients, packets routed by inner destination
- [x] 9.2 Allowed IPs: each peer may only send from its own VPN address

## CLI

Run with `go run ./cmd/netforge <command>`, or build once with `go build ./cmd/netforge`.

Available:

```bash
netforge echo
netforge dial [addr]
netforge proxy
netforge socks
netforge udpecho
netforge udpdial [addr]
netforge tunnel server
netforge tunnel client
netforge dns [addr]
netforge tun        # Linux only, sudo
netforge genkey
netforge pubkey <keyfile>
netforge vpn server <key> <peer.pub>=<vpn ip>...     # Linux only, sudo
netforge vpn client <host:port> <key> <peer.pub>     # Linux only, sudo
```
