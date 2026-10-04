# 2 — Transport Networking

> Concise revision sheet for sockets, ports, TCP, UDP, and transport-layer communication.

## Sockets

A **socket** is an operating-system networking object used by applications to communicate through the network stack.

```text
Application
    ↓
Socket API
    ↓
Socket
    ↓
OS TCP/IP stack
    ↓
Network
```

A socket is not the same thing as a port.

```text
Port   → transport endpoint number
Socket → OS object representing networking state/communication
```

### TCP server lifecycle

```text
socket
  ↓
bind
  ↓
listen
  ↓
accept
  ↓
read / write
  ↓
close
```

- `bind` associates a socket with a local IP/port.
- `listen` waits for incoming TCP connections.
- `accept` creates a connected socket for one client.
- The listening socket normally remains available for additional clients.

### TCP client lifecycle

```text
socket
  ↓
connect
  ↓
read / write
  ↓
close
```

### Binding addresses

```text
127.0.0.1:9000
→ loopback only / same machine
```

```text
0.0.0.0:9000
→ all suitable local IPv4 interfaces
```

Remote reachability can still depend on routing, firewall rules, NAT, and network topology.

---

## Ports and connection identity

Ports help the transport layer direct traffic to the correct endpoint/application.

```text
IP address → host/interface
Port       → transport endpoint
```

A client usually uses a temporary **ephemeral port**.

Example:

```text
192.168.1.20:53124
        ↕
203.0.113.10:443
```

A TCP connection is commonly identified by the 4-tuple:

```text
source IP
source port
destination IP
destination port
```

TCP and UDP port spaces are separate:

```text
TCP :9000 ≠ UDP :9000
```

---

## TCP essentials

**TCP = Transmission Control Protocol**

TCP provides a:

```text
connection-oriented
reliable
ordered
full-duplex
byte stream
```

### Three-way handshake

```text
Client                    Server

SYN --------------------->

    <---------------- SYN-ACK

ACK --------------------->
```

The operating system handles this when an application connects.

### TCP is a byte stream

TCP does not preserve application write boundaries.

If the sender does:

```text
Write("HELLO")
Write("WORLD")
```

the receiver may read:

```text
HELLOWORLD
```

or:

```text
HE
LLOWOR
LD
```

The byte order is preserved, but message boundaries are not.

Therefore application protocols need their own framing, such as:

```text
delimiters
length prefixes
fixed-size messages
protocol-specific framing
```

### Sequence numbers

TCP numbers bytes so it can:

```text
maintain ordering
detect missing data
detect duplicates
reassemble the byte stream
```

### ACKs

An acknowledgment communicates which data has been received and what byte is expected next.

Simplified:

```text
bytes 1-100 -------->

          <--------- ACK 101
```

### Retransmission

If TCP detects missing data, it can retransmit it.

```text
1-100      ✅
101-200    ❌ lost
201-300    ✅

TCP retransmits 101-200
```

### Reliability

TCP can recover from many cases of:

```text
loss
reordering
duplication
```

but a TCP connection can still fail because of outages, resets, crashes, timeouts, routing failures, etc.

### Flow control vs congestion control

```text
Flow control
→ protects the receiver from being overwhelmed
```

```text
Congestion control
→ adapts to network capacity/congestion
```

### Closing TCP

A graceful close commonly uses `FIN`.

```text
FIN → graceful shutdown of one sending direction
RST → abrupt reset/abort
```

`TIME_WAIT` is a TCP state that can remain briefly after closing so delayed packets from an old connection do not interfere with a newer one.

---

## UDP essentials

**UDP = User Datagram Protocol**

UDP provides:

```text
connectionless
message/datagram-oriented communication
```

It does **not** provide built-in guarantees for:

```text
delivery
ordering
automatic retransmission
TCP-style flow control
TCP-style congestion control
```

### UDP preserves datagram boundaries

```text
send [HELLO]
send [WORLD]

receive [HELLO]
receive [WORLD]
```

Unlike TCP, UDP does not merge these into one continuous byte stream.

### UDP lifecycle

UDP does not use the TCP-style:

```text
listen
accept
3-way handshake
```

A typical UDP application:

```text
socket
  ↓
bind if needed
  ↓
send datagram
receive datagram
```

A UDP socket can be "connected" to a default peer, but this does not create TCP-style reliability or a handshake.

### Why UDP is useful

UDP is useful when:

```text
latency matters
individual message boundaries matter
some loss is acceptable
the application wants custom transport behavior
```

Common examples include:

```text
DNS
real-time voice/video
online games
some VPN transports
QUIC
```

A protocol built on UDP can implement reliability itself.

---

## TCP vs UDP

| TCP | UDP |
|---|---|
| Connection-oriented | Connectionless |
| Reliable ordered stream | Datagrams with minimal guarantees |
| Byte-stream oriented | Message/datagram oriented |
| Retransmission built in | No automatic retransmission |
| Ordering built in | No ordering guarantee |
| Flow control | No built-in equivalent |
| Congestion control | No TCP-style built-in equivalent |
| More connection state | Less transport state |

Mental shortcut:

```text
TCP
→ "Give me a reliable ordered byte stream."
```

```text
UDP
→ "Send this datagram; the application decides what guarantees it needs."
```

---

## Multiplexing and demultiplexing

Many applications can share one network interface.

Outgoing:

```text
Browser ─┐
SSH ─────┼→ TCP/UDP → IP → network
Game ────┘
```

Incoming traffic is separated using protocol, ports, and socket state:

```text
network
   ↓
IP
   ↓
TCP / UDP
   ↓
port/socket lookup
   ↓
correct application
```

---

## Transport-layer mental model

```text
Application
    ↓
Socket
    ↓
TCP / UDP
    ↓
IP
    ↓
Routing
    ↓
Link layer
    ↓
Network
```

At the destination:

```text
Network
    ↓
Link layer
    ↓
IP
    ↓
TCP / UDP
    ↓
Port / socket
    ↓
Application
```
