# 🧪 netforge

**Learn networking fundamentals (theory) and build a proxy and an encrypted VPN from scratch in Go (practice).**

You start with a tiny TCP echo server and finish with a multi-peer VPN: X25519 handshake, AES-256-GCM, replay protection, NAT, full tunnel, DNS. Every stage pairs a lesson with working code that you run, observe on the wire, and break on purpose.

![Go](https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white)
![Platform](https://img.shields.io/badge/stages_1--5-Windows_·_macOS_·_Linux-555)
![Platform](https://img.shields.io/badge/stages_6--9-Linux_·_WSL2-555)
![Tests](https://img.shields.io/badge/end--to--end-46_checks-2ea44f)

---

## Contents

1. [What you will build](#what-you-will-build)
2. [How to study with this repo](#how-to-study-with-this-repo)
3. [Setup](#setup)
4. [Hands-on labs, stage by stage](#hands-on-labs-stage-by-stage)
5. [The Linux lab network](#the-linux-lab-network)
6. [Check everything works](#check-everything-works)
7. [CLI reference](#cli-reference)
8. [Project layout](#project-layout)
9. [Roadmap](#roadmap)
10. [Going further](#going-further)

---

## What you will build

```
TCP ─▶ HTTP proxy ─▶ SOCKS5 ─▶ UDP ─▶ Tunnel ─▶ TUN ─▶ VPN ─▶ Encryption ─▶ Multi-peer
```

| # | Stage | You build | Concepts you practice | Lesson |
|---|---|---|---|---|
| 1 | TCP | echo server, interactive client | sockets, handshake, byte stream, half-close (FIN), refused vs timeout | [2 · Transport](_networking-lessons/2-transport-networking.md) |
| 2 | HTTP proxy | forward proxy, plain HTTP + `CONNECT` | request framing, headers, HTTPS tunneling | [3 · HTTP, TLS, proxies](_networking-lessons/3-http-tls-proxies.md) |
| 3 | SOCKS5 | SOCKS5 server | binary protocols, big-endian, DNS leaks | [3 · HTTP, TLS, proxies](_networking-lessons/3-http-tls-proxies.md) |
| 4 | UDP | echo server, client that counts loss | datagrams, silent loss, ICMP errors, size limits | [2 · Transport](_networking-lessons/2-transport-networking.md) |
| 5 | Tunnel | point-to-point UDP tunnel | encapsulation, overhead, tunnel MTU, keepalives | [5 · Tunneling](_networking-lessons/5-tunneling.md) |
| 6 | TUN | a virtual network card in Go | raw IPv4, checksums, answering ping yourself | [6 · TUN / TAP](_networking-lessons/6-tun-tap.md) |
| 7 | VPN | VPN server and client | routing, IP forwarding, NAT, full tunnel, DNS through the VPN | [7 · VPN architecture](_networking-lessons/7-vpn-architecture.md) |
| 8 | Encryption | sealed packets and a handshake | AEAD, nonces, replay attacks, X25519, forward secrecy | [8 · Security & cryptography](_networking-lessons/8-security-cryptography.md) |
| 9 | Multi-peer | one server, several clients | peer table, peer routing, allowed IPs | [9 · Multi-peer](_networking-lessons/9-multi-peer-networking.md) |

Lessons [1 · Network fundamentals](_networking-lessons/1-network-fundamentals.md) and [4 · Advanced concepts](_networking-lessons/4-advanced-network-concepts.md) (namespaces, NAT, connection tracking) are used all along.

---

## How to study with this repo

1. **Read the lesson** of the stage in [`_networking-lessons/`](_networking-lessons/).
2. **Read the code.** Every networking idea in the code links its lesson section:
   ```go
   // Only now, after the packet passed every check, may it update this peer's endpoint:
   // first contact, back after a timeout, or roaming.
   // Theory: _networking-lessons/7-vpn-architecture.md#endpoint-changes--roaming
   ```
3. **Run the lab** below and watch what happens: logs, `ping`, `tcpdump`.
4. **Break it.** Change a value, remove a check, send a bad packet. Then explain what you see.
5. **See it.** Open [`docs/blueprint.html`](docs/blueprint.html) in a browser: an animated map of every scenario, with the Go line and the lesson for each packet.

💡 **Replay the build.** The history is one small step per commit, from the first echo server to allowed IPs:

```bash
git log --reverse --oneline      # the whole journey, step by step
git show <commit>                # what one step changed, and why
```

---

## Setup

| You need | For |
|---|---|
| [Go 1.27+](https://go.dev/dl/) | everything |
| `curl` | stages 2–3 |
| Linux with `sudo`, or **WSL2 Ubuntu** on Windows | stages 6–9 (TUN, namespaces) |
| `dnsutils` (`dig`), `tcpdump`, `iptables`, `python3` | stages 7–9 and the self-test |

```bash
git clone https://github.com/nixiz0/netforge.git
cd netforge
go test ./...                     # unit tests
go run ./cmd/netforge             # lists every command
```

<details>
<summary><b>Windows + WSL2 notes</b></summary>

- In PowerShell, type `curl.exe`, not `curl` (that is an alias for something else).
- Stages 6–9 run inside WSL2 Ubuntu. If `wsl` opens another distribution (Docker Desktop installs one), use `wsl -d Ubuntu`.
- Build the Linux binary with `CGO_ENABLED=0` (pure Go, no C toolchain needed) and `-buildvcs=false` (avoids a git ownership error on `/mnt/c`):
  ```bash
  CGO_ENABLED=0 go build -buildvcs=false -o /tmp/netforge ./cmd/netforge
  ```
- Install the lab tools once:
  ```bash
  sudo apt install -y dnsutils tcpdump iptables
  ```

</details>

---

## Hands-on labs, stage by stage

Stages 1–5 run on any OS, with `go run`. Stages 6–9 run on Linux / WSL2, as root, with the binary built above. Each lab says what to type and **what to look for**.

<details>
<summary><b>1 · TCP: echo server and client</b></summary>

```bash
go run ./cmd/netforge echo                      # terminal 1: server on 127.0.0.1:9000
go run ./cmd/netforge dial                      # terminal 2: type lines, they come back
```

End the input (Ctrl+D on Linux/macOS, Ctrl+Z then Enter on Windows): the client sends a FIN, the server finishes and closes.

```bash
go run ./cmd/netforge dial 127.0.0.1:9001       # closed port: "connection refused" (the kernel sent RST)
go run ./cmd/netforge dial 10.255.255.1:9000    # nobody there: "i/o timeout" after 5 s (silence)
```

👀 Refused is an answer; a timeout is the absence of one.

</details>

<details>
<summary><b>2 · HTTP proxy</b></summary>

```bash
go run ./cmd/netforge proxy                                  # terminal 1: 127.0.0.1:8080
curl -x http://127.0.0.1:8080 http://example.com/            # plain HTTP: the proxy reads the request
curl -x http://127.0.0.1:8080 https://example.com/           # HTTPS: CONNECT, then a blind encrypted tunnel
curl -x http://127.0.0.1:8080 http://nonexistent.invalid/    # 502 Bad Gateway
```

👀 The proxy log shows the request line for HTTP, but only `CONNECT example.com:443` for HTTPS: it never sees inside TLS.

</details>

<details>
<summary><b>3 · SOCKS5</b></summary>

```bash
go run ./cmd/netforge socks                                       # terminal 1: 127.0.0.1:1080
curl --socks5-hostname 127.0.0.1:1080 https://example.com/        # the proxy resolves the name
curl --socks5 127.0.0.1:1080 https://example.com/                 # curl resolves it first: a DNS leak
```

👀 The log says who resolved DNS: `domain name: the proxy resolves it` vs `IP address: the client resolved it`.

</details>

<details>
<summary><b>4 · UDP</b></summary>

```bash
go run ./cmd/netforge udpecho                       # terminal 1: 127.0.0.1:9000/udp
go run ./cmd/netforge udpdial                       # terminal 2: one line = one datagram
go run ./cmd/netforge udpdial 10.255.255.1:9000     # nobody answers: loss is silent, only counted
go run ./cmd/netforge udpdial 127.0.0.1:9009        # closed port: an ICMP error shows up on a later read
```

In `udpdial`, type `+1472`, `+65507`, `+65508` to send datagrams of that size.

👀 1472 bytes fit one Ethernet frame; 65508 fails with "message too long"; nothing tells you a datagram was lost.

</details>

<details>
<summary><b>5 · Tunnel</b></summary>

```bash
go run ./cmd/netforge udpecho                       # terminal 1: the target
go run ./cmd/netforge tunnel server                 # terminal 2: far end, 127.0.0.1:7000/udp
go run ./cmd/netforge tunnel client                 # terminal 3: entrance, 127.0.0.1:5300/udp
go run ./cmd/netforge udpdial 127.0.0.1:5300        # terminal 4: talk to udpecho THROUGH the tunnel
```

Type `+1468`, then `+1469`.

👀 The logs show each datagram growing by 4 bytes (`encapsulated 22 bytes (50 on the wire)`). 1468 bytes is exactly 1500 on the wire; 1469 is dropped: that is the tunnel MTU.

</details>

<details>
<summary><b>6 · TUN: a virtual network card (Linux)</b></summary>

```bash
sudo /tmp/netforge tun                                              # terminal 1: creates nf0
sudo ip addr add 10.10.0.1/24 dev nf0 && sudo ip link set nf0 up mtu 1468   # terminal 2
ping -c 3 10.10.0.2                                                 # answered by netforge itself
curl -m 2 http://10.10.0.2/                                         # a TCP SYN shows up too
```

👀 netforge decodes each raw IPv4 packet (`IPv4 10.10.0.1 -> 10.10.0.2 ICMP ttl=64, 84 bytes`) and writes a ping reply back with correct checksums.

</details>

<details>
<summary><b>7 · VPN: routing, NAT, full tunnel, DNS (Linux)</b></summary>

Keys and lab, once (see [the lab network](#the-linux-lab-network)):

```bash
/tmp/netforge genkey > /tmp/server.key && /tmp/netforge pubkey /tmp/server.key > /tmp/server.pub
/tmp/netforge genkey > /tmp/client.key && /tmp/netforge pubkey /tmp/client.key > /tmp/client.pub
sudo bash scripts/lab.sh up
```

```bash
sudo ip netns exec nfpeer /tmp/netforge vpn server /tmp/server.key /tmp/client.pub=10.10.0.1   # terminal 1
sudo /tmp/netforge vpn client 192.168.100.2:7000 /tmp/client.key /tmp/server.pub              # terminal 2
sudo bash scripts/lab.sh tun                       # terminal 3: addresses on both nf0
ping -c 3 10.10.0.2                                # the server, through the VPN
```

| Step | Commands | 👀 What to look for |
|---|---|---|
| Server as router | `sudo bash scripts/lab.sh route && sudo bash scripts/lab.sh forward`, then `ping -c 3 203.0.113.10` | a host behind the server answers; `ttl=63` = one router hop |
| NAT | `sudo bash scripts/lab.sh nat`, then the same ping | the host now sees `203.0.113.1`, never your `10.10.0.1` |
| Full tunnel | `sudo bash scripts/lab.sh full`, then `ip route get 1.1.1.1` | everything goes into nf0, except the tunnel itself (WSL has no internet until `full-off`) |
| DNS leak | `sudo bash scripts/lab.sh dns`, then `sudo bash scripts/lab.sh full-off` | `LEAK`: your resolver is outside the tunnel, even in full mode |
| DNS inside the VPN | `sudo ip netns exec nfpeer /tmp/netforge dns :53`, then `dig @10.10.0.2 host.lab` and `dig @192.168.100.2 host.lab` | same answer, but the server log shows the first came through the tunnel (`10.10.0.1`), the second beside it |

</details>

<details>
<summary><b>8 · Encryption: AEAD, replay, handshake (Linux)</b></summary>

With the VPN from stage 7 running:

```bash
sudo tcpdump -X -c 4 -ni veth0 udp port 7000 &     # watch the wire...
ping -c 2 10.10.0.2                                # ...only ciphertext: 84-byte pings travel as 116 bytes
sudo python3 scripts/replay.py                     # replay attack: the copy is dropped
```

Restart the client with a key the server does not know:

```bash
/tmp/netforge genkey > /tmp/bad.key
sudo /tmp/netforge vpn client 192.168.100.2:7000 /tmp/bad.key /tmp/server.pub
```

👀 Server log: `replayed packet`, then `drop 72 bytes … authentication failed`. Restart the server: after 15 s the client logs `tunnel down`, runs a new handshake, and traffic resumes with fresh keys.

</details>

<details>
<summary><b>9 · Multi-peer: two clients, allowed IPs (Linux)</b></summary>

```bash
/tmp/netforge genkey > /tmp/client2.key && /tmp/netforge pubkey /tmp/client2.key > /tmp/client2.pub
sudo bash scripts/lab.sh up2                       # client 2: namespace nfc2 + a cable to the server
sudo ip netns exec nfpeer /tmp/netforge vpn server /tmp/server.key /tmp/client.pub=10.10.0.1 /tmp/client2.pub=10.10.0.3   # terminal 1, restarted
sudo ip netns exec nfc2 /tmp/netforge vpn client 192.168.101.2:7000 /tmp/client2.key /tmp/server.pub                    # terminal 4
sudo bash scripts/lab.sh tun && sudo bash scripts/lab.sh tun2 && sudo bash scripts/lab.sh forward
ping -c 3 10.10.0.3                                # client 1 -> server -> client 2
sudo bash scripts/lab.sh spoof                     # client 2 pretends to be 10.10.0.1
```

👀 Each ping crosses the server twice: decrypted from client 1, re-encrypted for client 2. The spoofed ping is dropped: `source not allowed`.

Clean up when you are done: `sudo bash scripts/lab.sh down`.

</details>

---

## The Linux lab network

[`scripts/lab.sh`](scripts/lab.sh) builds four "machines" inside one Linux box with network namespaces, wired by virtual Ethernet cables:

```
 client 1 (root namespace)         VPN server (nfpeer)                host on the "internet" (nfnet)
 nf0   10.10.0.1                   nf0   10.10.0.2
 veth0 192.168.100.1 ───────────── veth1 192.168.100.2
                                   veth2 203.0.113.1 ──────────────── veth3 203.0.113.10
                                   veth4 192.168.101.2
 client 2 (nfc2)                         │
 nf0   10.10.0.3                         │
 veth5 192.168.101.1 ────────────────────┘

 10.10.0.0/24 = addresses inside the VPN (nf0)     192.168.10x.0/24 = the network the tunnel travels on
```

| `lab.sh` | Does |
|---|---|
| `up` / `down` | create / remove the namespaces and cables |
| `tun`, `tun2` | give the nf0 interfaces their addresses (MTU 1440) |
| `route`, `forward` | split tunnel and return route; IP forwarding on the server |
| `nat` | the server masquerades the VPN subnet |
| `full`, `full-off` | send all IPv4 into the tunnel, and back |
| `dns` | show whether DNS goes through the VPN or leaks |
| `up2`, `spoof` | add client 2; make it spoof client 1's address |

---

## Check everything works

One command runs every stage on real traffic and checks logs, pings and `tcpdump` captures (Linux / WSL2, about 2 minutes):

```bash
sudo bash scripts/e2e.sh          # 46 checks; logs kept in /tmp/netforge-e2e
```

---

## CLI reference

`go run ./cmd/netforge <command>`, or build once with `go build ./cmd/netforge`.

| Command | Does |
|---|---|
| `echo` | TCP echo server on `127.0.0.1:9000` |
| `dial [addr]` | interactive TCP client (default `127.0.0.1:9000`) |
| `proxy` | HTTP / HTTPS forward proxy on `127.0.0.1:8080` |
| `socks` | SOCKS5 proxy on `127.0.0.1:1080` |
| `udpecho` | UDP echo server on `127.0.0.1:9000` |
| `udpdial [addr]` | UDP client: one line = one datagram, `+N` = N-byte probe |
| `tunnel server` / `tunnel client` | the two ends of the plain UDP tunnel |
| `dns [addr]` | tiny authoritative DNS server (`host.lab`, `vpn.lab`), default `127.0.0.1:5353` |
| `tun` | 🐧 create `nf0`, decode IPv4, answer ping |
| `genkey` / `pubkey <keyfile>` | X25519 private key / its public key |
| `vpn server <key> <peer.pub>=<vpn ip>...` | 🐧 VPN server with one or more peers |
| `vpn client <host:port> <key> <server.pub>` | 🐧 VPN client |

🐧 = Linux only, run with `sudo`.

---

## Project layout

```
cmd/netforge/          the binary: one subcommand per feature
internal/              one package per feature: echo, dial, proxy, socks, relay, resolve, tunnel, tun, dns
_networking-lessons/   the theory, one file per topic
scripts/               lab.sh (lab network), e2e.sh (self-test), replay.py (replay attack)
docs/blueprint.html    interactive map of every scenario
ARCHITECTURE.md        internals and conventions, for contributors
```

---

## Roadmap

All 9 stages are done. Each sub-step is one commit.

<details>
<summary><b>Every step, in order</b></summary>

### 1 — TCP
- [x] 1.1 Echo server, one client at a time
- [x] 1.2 One goroutine per client
- [x] 1.3 Split into files, `netforge echo` command
- [x] 1.4 Minimal client: dial, send, read
- [x] 1.5 Byte stream: one `Read` is not one message
- [x] 1.6 Interactive client, both directions at once
- [x] 1.7 Half-close (FIN)
- [x] 1.8 Failures: refused vs timeout, DNS

### 2 — HTTP proxy
- [x] 2.1 TCP forwarder: accept, dial a fixed target, relay both ways
- [x] 2.2 Read the HTTP request (request line, headers, `\r\n` framing)
- [x] 2.3 Plain HTTP proxy: dial the destination from the request, forward it
- [x] 2.4 `CONNECT`: HTTPS through the proxy

### 3 — SOCKS5
- [x] 3.1 Greeting: method negotiation, binary framing with `io.ReadFull`
- [x] 3.2 Read the CONNECT request (address type, big-endian port)
- [x] 3.3 Share the relay: move it to `internal/relay`
- [x] 3.4 Connect, reply, relay: a working SOCKS5 proxy
- [x] 3.5 DNS: client-side vs proxy-side resolution
- [x] 3.6 Explicit DNS lookups in both proxies, logged with timing

### 4 — UDP
- [x] 4.1 UDP echo server: one socket, no handshake, sender address per datagram
- [x] 4.2 UDP client: one write = one datagram, boundaries kept
- [x] 4.3 Loss is silent: read deadlines, closed ports
- [x] 4.4 Datagram size: limits and MTU (`+N` probes)

### 5 — Tunnel
- [x] 5.1 Tunnel packet format: 4-byte header + payload, encode/decode, first tests
- [x] 5.2 Tunnel server: decapsulate, forward to the inner target, wrap replies
- [x] 5.3 Tunnel client: local entrance, end to end udpdial → tunnel → udpecho
- [x] 5.4 Overhead and tunnel MTU (1468 = 1500 − 32)
- [x] 5.5 Keepalives and tunnel state (5 s keepalive, 15 s peer timeout)

### 6 — TUN
- [x] 6.1 Open a TUN interface (`nf0`) and log the raw IP packets routed into it
- [x] 6.2 Decode the IPv4 header (tested parser)
- [x] 6.3 Write path: answer ping from userspace (ICMP echo reply + checksum)
- [x] 6.4 TUN ⇄ tunnel: IP packets through the tunnel between two network namespaces

### 7 — VPN
- [x] 7.1 `vpn server` / `vpn client`: the server learns its client, keepalives, roaming
- [x] 7.2 The server as a router: IP forwarding, a machine behind the server, return route
- [x] 7.3 NAT on the server
- [x] 7.4 Full tunnel without routing the tunnel into itself
- [x] 7.5 See the DNS leak: the resolver's route bypasses the full tunnel
- [x] 7.6 A DNS server inside the VPN (`netforge dns`, reached at the server's inner address)

### 8 — Encryption
- [x] 8.1 Seal every VPN packet with AEAD (AES-256-GCM, shared key): confidentiality, integrity, no more hijack
- [x] 8.2 Replay protection: a sliding window over packet counters
- [x] 8.3 Key exchange: X25519 key pairs, a handshake, fresh session keys (forward secrecy)

### 9 — Multi-peer
- [x] 9.1 Peer table: one server, two clients, packets routed by inner destination
- [x] 9.2 Allowed IPs: each peer may only send from its own VPN address

</details>

---

## Going further

Ideas to extend the project on your own:

- Run the VPN server on a real VPS and connect from home.
- Carry IPv6 inside the tunnel.
- Rekey sessions periodically.
- Allow whole networks behind a peer (site-to-site), with longest-prefix match between peers.
- Find peers with a receiver index instead of trying each key, as WireGuard does.

> ⚠️ **Educational project.** netforge shows how a VPN works; it is not audited and not meant to protect real traffic. For real use, choose WireGuard.
