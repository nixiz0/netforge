# 7 — VPN Architecture

> Concise revision sheet for full/split tunneling, VPN routing, forwarding, NAT, DNS, MTU, kill switches, and VPN endpoints.

---

# VPN Architecture

A practical VPN combines several ideas:

```text
TUN
+ routing
+ tunneling
+ transport
+ security
=
VPN architecture
```

A useful high-level model is:

```text
Application
   ↓
OS routing
   ↓
tun0
   ↓
VPN client
   ↓
encrypted tunnel
   ↓
Internet
   ↓
VPN server
   ↓
destination network
```

---

# Inner vs Outer Addresses

A VPN usually has two different address worlds.

## Inner / VPN addresses

Example:

```text
Client VPN IP:
10.8.0.2

Server VPN IP:
10.8.0.1
```

These belong to the virtual VPN network.

## Outer / real-network addresses

Example:

```text
Client public/NAT IP:
198.51.100.20

VPN server public IP:
203.0.113.50
```

These are used to carry the tunnel over the real network.

Mental model:

```text
Inner addresses
→ virtual VPN network

Outer addresses
→ underlying real network
```

---

# Full Tunnel

A **full-tunnel VPN** routes nearly all ordinary traffic through the VPN.

Conceptually:

```text
Application
   ↓
default route
   ↓
tun0
   ↓
VPN server
   ↓
Internet
```

Typical route idea:

```text
0.0.0.0/0 → tun0
```

For IPv6:

```text
::/0 → VPN interface
```

The destination website usually sees the VPN server's public-facing address rather than the client's original public-facing address.

---

# Split Tunnel

A **split tunnel** routes only selected destinations through the VPN.

Example:

```text
10.20.0.0/16 → tun0
10.30.0.0/16 → tun0

everything else
→ normal gateway
```

Mental shortcut:

```text
Full tunnel
→ nearly all traffic through VPN

Split tunnel
→ selected traffic through VPN
```

---

# VPN Server as Router

A VPN server often needs to forward packets between interfaces.

Example:

```text
tun0
 ↓
VPN server routing
 ↓
eth0
 ↓
Internet
```

The server must distinguish:

```text
traffic for itself
```

from:

```text
traffic passing through it
```

---

# IP Forwarding

**IP forwarding** allows the OS to route packets between interfaces.

Example:

```text
tun0
 ↕
Linux routing
 ↕
eth0
```

Without forwarding, the server may receive tunneled traffic but not pass it onward.

Mental model:

```text
IP forwarding
→ let the OS route packets through the machine
```

---

# NAT on a VPN Server

A VPN client may use a private VPN address such as:

```text
10.8.0.2
```

Public Internet routers do not normally know how to route replies directly to that private address.

A VPN server may therefore use NAT:

```text
VPN client
10.8.0.2

   ↓ tunnel

VPN server

   ↓ NAT

VPN server public IP
203.0.113.50

   ↓

Internet
```

The public destination sees traffic as coming from the VPN server's public address.

Return traffic is translated back and sent through the tunnel.

---

# Routing Instead of NAT

NAT is not always required.

If other routers know:

```text
10.8.0.0/24
→ VPN server
```

then ordinary routing can carry return traffic correctly.

So:

```text
NAT
→ one solution

Correct routes
→ another solution
```

---

# Avoid Routing the Tunnel into Itself

Suppose the VPN server endpoint is:

```text
203.0.113.50
```

and the client installs:

```text
0.0.0.0/0 → tun0
```

Traffic used to reach the VPN server itself could also be sent into the tunnel.

That would create a recursive routing problem.

So a more specific route is usually kept:

```text
203.0.113.50
→ physical gateway
```

while:

```text
0.0.0.0/0
→ tun0
```

Longest-prefix matching ensures the specific server route wins.

---

# DNS Through a VPN

A VPN may carry normal traffic while DNS still uses a local ISP or local-network resolver.

Example:

```text
web traffic → VPN
DNS queries  → local network
```

This can expose domain lookups outside the intended VPN path.

This is commonly called a:

```text
DNS leak
```

A VPN may configure DNS so that DNS queries follow the intended VPN path.

---

# Split DNS

Sometimes only selected domains should use VPN DNS.

Example:

```text
company.internal
→ corporate DNS through VPN
```

while:

```text
example.com
→ normal DNS resolver
```

This is called:

```text
split DNS
```

---

# VPN MTU

VPN encapsulation adds overhead.

```text
inner packet
+
tunnel headers
+
transport headers
+
encryption/authentication overhead
=
larger outer packet
```

So the TUN interface often uses a smaller MTU than the physical interface.

Example:

```text
Physical path MTU = 1500
VPN overhead      = 80

tun0 MTU          ≈ 1420
```

Bad MTU settings can cause symptoms such as:

```text
small requests work
large transfers fail or hang
some websites load partially
```

---

# Kill Switch

A **kill switch** prevents protected traffic from escaping outside the VPN if the VPN connection fails.

Conceptually:

```text
VPN UP
→ allow traffic through VPN
```

```text
VPN DOWN
→ block protected traffic
```

A kill switch is mainly a routing/firewall policy.

It is not encryption itself.

---

# VPN Endpoint vs VPN Address

## Endpoint

The endpoint is the real network address used to reach the peer.

Example:

```text
203.0.113.50:51820
```

## VPN address

The VPN address is the virtual address inside the VPN network.

Example:

```text
10.8.0.1
```

Mental shortcut:

```text
Endpoint
→ real IP:port used to reach peer

VPN address
→ virtual address inside VPN
```

---

# Endpoint Changes / Roaming

A mobile client may move between networks:

```text
Home Wi-Fi
↓
4G/5G
↓
Coffee-shop Wi-Fi
```

Its real endpoint can change.

Example:

```text
old endpoint:
198.51.100.20:50000

new endpoint:
192.0.2.40:62000
```

More advanced VPNs can update the peer's known endpoint.

This is often called:

```text
roaming
```

---

# VPN Routing Policy

A VPN needs to decide which destination prefixes belong in the tunnel.

Example split policy:

```text
10.20.0.0/16
→ VPN

172.16.50.0/24
→ VPN

everything else
→ normal Internet
```

Full tunnel:

```text
0.0.0.0/0
→ VPN
```

So VPN architecture is deeply tied to routing.

---

# Full VPN Data Path

## Send

```text
Application
   ↓
TCP / UDP
   ↓
Inner IP packet
   ↓
Routing table
   ↓
tun0
   ↓
VPN client
   ↓
Encrypt + authenticate
   ↓
Encapsulate
   ↓
UDP/TCP transport
   ↓
Outer IP
   ↓
Physical network
   ↓
VPN server
```

## Server side

```text
Receive outer packet
   ↓
Authenticate
   ↓
Decrypt
   ↓
Decapsulate
   ↓
Recover inner packet
   ↓
Route / NAT
   ↓
Destination
```

Return traffic follows the reverse process.

---

# What Makes a VPN More Than a Tunnel?

A basic tunnel mainly provides:

```text
encapsulation
```

A practical VPN adds:

```text
virtual interface
routing
peer identity
authentication
encryption
integrity
key management
replay protection
NAT/firewall handling
```

Useful high-level definition:

```text
VPN
→ secure routed tunnel system
```

---

# One-Minute Cheat Sheet

```text
Full tunnel
→ nearly all traffic through VPN
```

```text
Split tunnel
→ selected routes through VPN
```

```text
VPN server
→ often acts as a router
```

```text
IP forwarding
→ lets packets pass between interfaces
```

```text
NAT
→ often used when VPN clients access the public Internet
```

```text
Endpoint
→ real IP:port used to reach a VPN peer
```

```text
VPN address
→ virtual address inside the VPN
```

```text
DNS leak
→ DNS bypasses intended VPN path
```

```text
Split DNS
→ selected domains use selected resolvers
```

```text
Kill switch
→ blocks protected traffic if VPN fails
```

```text
VPN MTU
→ often smaller because tunneling adds overhead
```

```text
VPN
→ secure routed tunnel system
```
