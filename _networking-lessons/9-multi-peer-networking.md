# 9 — Multi-Peer Networking

> Concise revision sheet for peer identity, endpoints, VPN addresses, allowed networks, peer routing, roaming, topology, relays, and control/data planes.

---

# Peer Identity, VPN Address, Endpoint

```text
Identity
→ who the peer is

VPN address
→ virtual network address

Endpoint
→ current real IP:port
```

Example:

```text
Identity: Public Key A
VPN IP: 10.8.0.5
Endpoint: 198.51.100.20:51820
```

A peer's endpoint can change while its identity remains the same.

---

# Peer Table

A multi-peer VPN may track:

```text
identity
VPN address
allowed networks
current endpoint
session keys
packet counters
replay state
last activity
keepalive state
```

---

# Allowed Networks

Allowed networks map destinations to peers.

```text
10.8.0.2/32
→ Peer A

10.20.0.0/16
→ Peer B
```

When a packet targets:

```text
10.20.5.10
```

the VPN selects Peer B.

Allowed networks can also act as authorization:

```text
Peer A allowed source:
10.8.0.2/32
```

If Peer A claims another source address, the packet can be rejected.

---

# Peer Routing

Conceptual send decision:

```text
destination IP
↓
route lookup
↓
peer selection
↓
peer security/session state
↓
peer endpoint
```

Longest-prefix matching still applies.

Example:

```text
10.0.0.0/8     → Peer A
10.20.0.0/16   → Peer B
```

Destination:

```text
10.20.5.1
```

uses Peer B because `/16` is more specific.

---

# Hub-and-Spoke

```text
        Peer A
          |
Peer B -- Hub -- Peer C
          |
        Peer D
```

Traffic passes through a central hub.

Advantages:

```text
simple
central control
simpler routing
```

Trade-off:

```text
central dependency
```

---

# Mesh

Peers communicate directly:

```text
Peer A -------- Peer B
  |              |
Peer C -------- Peer D
```

Advantages:

```text
direct paths
potentially lower latency
less central traffic
```

More complexity:

```text
NAT traversal
peer discovery
endpoint changes
route management
key management
```

---

# Hybrid / Relay Fallback

Direct path when possible:

```text
Peer A ←──── direct encrypted path ────→ Peer B
```

Fallback when direct connectivity fails:

```text
Peer A
  ↓
Relay
  ↓
Peer B
```

The relay can forward encrypted traffic without being the plaintext endpoint.

---

# Roaming

A known peer may move between networks:

```text
old endpoint:
198.51.100.20:50000

new endpoint:
203.0.113.45:61000
```

If packets authenticate as the same peer, the VPN can update the endpoint.

```text
Roaming
→ authenticated peer changes real endpoint
```

Endpoint updates should happen only after authentication.

---

# Keepalives

Peers behind NAT/firewalls may need periodic keepalives to preserve:

```text
NAT mappings
firewall state
reachability
```

---

# Per-Peer Security State

Different peers should normally have separate session/security state.

```text
Peer A ↔ Peer B
session AB

Peer A ↔ Peer C
session AC
```

Avoid relying on one global session key for all peers.

---

# Route Conflicts

If multiple peers claim the same network:

```text
Peer A → 10.20.0.0/16
Peer B → 10.20.0.0/16
```

the VPN needs a clear policy.

Possible mechanisms:

```text
unique ownership
longest-prefix match
priority
administrative policy
```

---

# Peer Discovery

Peers may be discovered through:

```text
static configuration
manual configuration
coordination server
directory/control plane
```

Discovery/configuration is separate from packet forwarding.

---

# Control Plane vs Data Plane

## Control Plane

Decides:

```text
which peers exist
which identities are trusted
which routes belong to which peers
where peers are currently reachable
configuration
```

```text
Control plane
→ decides
```

## Data Plane

Actually forwards traffic:

```text
TUN packet
↓
peer lookup
↓
encrypt
↓
send
```

```text
Data plane
→ forwards
```

---

# Coordination Server

A coordination server may tell peers:

```text
Peer B exists
Peer B public identity/key = ...
Peer B endpoint = ...
Peer B routes = ...
```

But traffic may still flow directly:

```text
Peer A ←──── encrypted direct tunnel ────→ Peer B
```

Important:

```text
Coordination server
≠
necessarily traffic relay
```

---

# Relay Server

If direct peer connectivity fails:

```text
Peer A
  ↓
Relay
  ↓
Peer B
```

The relay is mainly a transport intermediary.

End-to-end encryption can remain between the peers.

---

# Send Flow

```text
Application
   ↓
OS routing
   ↓
TUN
   ↓
VPN process
   ↓
destination lookup
   ↓
select peer
   ↓
peer session/security state
   ↓
encrypt/authenticate
   ↓
peer endpoint
   ↓
Internet
```

---

# Receive Flow

```text
Internet
   ↓
VPN socket
   ↓
identify/authenticate peer
   ↓
replay check
   ↓
decrypt
   ↓
validate source address
   ↓
write inner packet to TUN
```

---

# One-Minute Cheat Sheet

```text
Peer identity
→ who the peer is
```

```text
VPN address
→ virtual network address
```

```text
Endpoint
→ real IP:port
```

```text
Allowed networks
→ routing + authorization
```

```text
Peer routing
→ destination prefix → peer
```

```text
Roaming
→ same authenticated peer, new endpoint
```

```text
Hub-and-spoke
→ central hub
```

```text
Mesh
→ direct peer-to-peer
```

```text
Relay
→ fallback transport
```

```text
Control plane
→ decides peers/routes/configuration
```

```text
Data plane
→ moves encrypted packets
```

```text
Per-peer security state
→ separate sessions/keys per peer
```
