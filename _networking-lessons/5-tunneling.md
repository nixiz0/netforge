# 5 — Tunneling

> Concise revision sheet for encapsulation, tunnel endpoints, UDP/TCP tunneling, MTU overhead, keepalives, and tunnel routing.

---

# Tunneling Essentials

## What is a tunnel?

A **tunnel** carries traffic inside another protocol or connection.

```text
original traffic
      ↓
encapsulate
      ↓
send through tunnel
      ↓
decapsulate
      ↓
deliver original traffic
```

Mental model:

```text
Tunnel
→ carry one kind of traffic inside another transport/protocol
```

---

# Encapsulation

A tunnel adds another wrapper around existing traffic.

Example:

```text
Original IP packet
        ↓
Tunnel header
        ↓
UDP
        ↓
Outer IP packet
```

Conceptually:

```text
[ Outer IP
    [ UDP
        [ Tunnel
            [ Original IP packet ]
        ]
    ]
]
```

The original packet becomes payload of the outer protocol.

---

# Inner vs Outer Packet

## Inner packet

The **inner packet** is the original traffic being carried.

Example:

```text
10.10.0.2 → 10.10.0.5
```

## Outer packet

The **outer packet** is used to cross the real underlying network.

Example:

```text
198.51.100.10 → 203.0.113.20
```

Conceptually:

```text
Outer packet
┌──────────────────────────────┐
│ Internet routing addresses   │
│                              │
│  Inner packet                │
│  ┌────────────────────────┐  │
│  │ VPN/private addresses  │  │
│  └────────────────────────┘  │
└──────────────────────────────┘
```

Internet routers normally route using the **outer IP header**.

---

# Tunnel Endpoints

A basic tunnel has two endpoints.

```text
Tunnel Client
      |
      | encapsulated traffic
      |
      v
Tunnel Server
```

One side encapsulates; the other decapsulates.

---

# Why Tunnels Are Useful

Tunnels can carry traffic through a network that would not normally route that traffic directly.

Example:

```text
Private LAN A
10.0.0.0/24

      ↓ tunnel

Public Internet

      ↓ tunnel

Private LAN B
10.20.0.0/24
```

The public Internet routes the outer packets.

The private inner packets are carried inside them.

---

# Tunnel Does Not Mean Encryption

These are separate concepts.

```text
Tunneling
→ encapsulation
```

```text
Encryption
→ confidentiality
```

Possible combinations:

```text
Tunnel + no encryption
Tunnel + encryption
```

A VPN commonly combines both, but they are not the same thing.

---

# Tunnel vs Proxy

## Proxy

```text
Application
   ↓
Proxy
   ↓
Destination
```

A proxy typically works at the application/connection level.

## Tunnel

```text
packet/data
   ↓
encapsulate
   ↓
transport
   ↓
decapsulate
```

A network tunnel can carry lower-level traffic without needing to understand the application protocol.

---

# Tunnel Transport

A tunnel still needs a transport across the underlying network.

Examples:

```text
Tunnel payload
   ↓
UDP
   ↓
IP
```

or:

```text
Tunnel payload
   ↓
TCP
   ↓
IP
```

Both are possible, but they have different trade-offs.

---

# UDP Tunneling

UDP is often attractive for tunnels because it gives simple datagrams without TCP's reliable byte-stream behavior.

```text
Original packet
      ↓
Tunnel protocol
      ↓
UDP datagram
      ↓
Internet
```

This lets the tunnel protocol decide how to handle:

```text
loss
ordering
retransmission
keepalives
encryption
```

Mental model:

```text
UDP tunnel
→ more control for the tunnel protocol
```

---

# TCP Tunneling

Traffic can also be tunneled over TCP.

```text
Original traffic
      ↓
Tunnel
      ↓
TCP
      ↓
Internet
```

This can work, but tunneling TCP inside TCP has an important drawback.

---

# TCP-over-TCP Problem

Example:

```text
Inner TCP connection
        ↓
TCP tunnel
        ↓
Internet
```

Now two TCP layers may independently perform:

```text
retransmission
ordering
congestion control
```

If the outer TCP stream loses data, it waits for retransmission and preserves ordering.

At the same time, the inner TCP connection may also react to delay/loss.

This interaction can cause poor performance, especially on lossy networks.

Mental model:

```text
TCP-over-TCP
→ two reliability/congestion systems stacked together
→ can interfere with each other
```

---

# Head-of-Line Blocking

TCP must present bytes in order.

Suppose:

```text
A ✅
B ❌ lost
C ✅
D ✅
```

TCP cannot normally deliver `C` and `D` to the application before `B` is recovered.

```text
A
[wait for B]
C
D
```

This is **head-of-line blocking**.

If many independent flows are tunneled through one TCP connection, one lost section of the outer stream can delay unrelated tunneled traffic.

UDP does not impose the same ordered-stream behavior.

---

# Tunnel Overhead

Encapsulation adds bytes.

Possible overhead:

```text
outer IP header
UDP/TCP header
tunnel header
encryption/authentication overhead
```

Example:

```text
1500-byte inner packet
+
70 bytes tunnel overhead
=
1570 bytes
```

If the path MTU is 1500, this is too large.

---

# Tunnel MTU

Because of encapsulation overhead, tunnel interfaces often use a smaller MTU.

Example:

```text
Physical MTU = 1500

Tunnel overhead ≈ 80

Tunnel MTU ≈ 1420
```

Mental model:

```text
Physical/path MTU
-
Tunnel overhead
=
usable inner packet size
```

The exact value depends on the tunnel protocol and path.

---

# Keepalives

NAT mappings and firewall state can expire when a connection is idle.

A tunnel may periodically send small packets:

```text
keepalive
keepalive
keepalive
```

Purpose:

```text
Keepalive
→ maintain NAT/firewall/tunnel state during idle periods
```

Keepalives are especially useful for peers behind NAT.

---

# Tunnel State

A tunnel implementation may track:

```text
remote endpoint
peer identity
last activity
keys
sequence numbers
MTU
connection/session state
```

This becomes more important in VPN and multi-peer systems.

---

# Point-to-Point Tunnel

Simplest model:

```text
Client A
   ═════════ tunnel ═════════
                         Server B
```

Traffic enters one endpoint and exits the other.

---

# Site-to-Site Tunnel

A tunnel can connect whole networks.

```text
Office A LAN
10.0.0.0/24
    |
Gateway A
    ═════ tunnel ═════
                  Gateway B
                      |
                Office B LAN
                10.20.0.0/24
```

Devices behind the gateways may not need to know the tunnel exists.

---

# Tunnel Routing

A tunnel becomes useful when the routing table directs selected traffic into it.

Example route:

```text
10.20.0.0/24
→ tunnel0
```

Then:

```text
Application wants 10.20.0.8
        ↓
routing table
        ↓
tunnel0
        ↓
encapsulation
        ↓
underlying network
```

So:

```text
Routing
→ decides which traffic enters the tunnel
```

---

# Full Tunnel Flow

Example inner packet:

```text
10.0.0.2 → 10.20.0.5
```

Flow:

```text
Application
   ↓
OS creates IP packet
   ↓
Routing table chooses tunnel
   ↓
Tunnel receives inner packet
   ↓
Encapsulation
   ↓
Outer UDP/TCP + IP packet
   ↓
Internet
   ↓
Remote tunnel endpoint
   ↓
Decapsulation
   ↓
Original inner IP packet
   ↓
Routing
   ↓
Destination
```

---

# One-Minute Cheat Sheet

```text
Tunnel
→ carry traffic inside another protocol/connection
```

```text
Encapsulation
→ wrap original traffic inside outer traffic
```

```text
Inner packet
→ original traffic being carried
```

```text
Outer packet
→ packet used to cross the real network
```

```text
Tunnel ≠ encryption
```

```text
UDP tunnel
→ gives tunnel protocol more control over loss/order/reliability
```

```text
TCP-over-TCP
→ can suffer because two TCP reliability systems interact
```

```text
Head-of-line blocking
→ missing TCP data delays later bytes
```

```text
Tunnel overhead
→ reduces usable MTU
```

```text
Keepalive
→ helps maintain NAT/firewall/tunnel state
```

```text
Routing table
→ decides which traffic enters the tunnel
```
