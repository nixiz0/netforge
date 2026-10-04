# 4 — Advanced Network Concepts

> Concise revision sheet for IPv6, stateful firewalls, connection tracking, NAT traversal, and Linux virtual networking concepts.

---

# IPv6 Essentials

## Why IPv6 exists

IPv4 uses 32-bit addresses:

```text
192.168.1.50
```

IPv6 uses 128-bit addresses:

```text
2001:db8:1234:5678::10
```

Main idea:

```text
IPv6
→ vastly larger address space
```

## IPv6 notation

IPv6 uses hexadecimal groups.

Full form:

```text
2001:0db8:0000:0000:0000:0000:0000:0001
```

Compressed:

```text
2001:db8::1
```

`::` replaces one or more consecutive groups of zeros.

## IPv6 prefixes

IPv6 still uses CIDR prefixes.

Example:

```text
2001:db8:1234:5678::/64
```

Meaning:

```text
first 64 bits
→ network prefix

remaining 64 bits
→ interface/address portion
```

`/64` is very common on IPv6 LANs.

## IPv6 neighbor discovery

IPv4:

```text
next-hop IPv4
↓
ARP
↓
MAC address
```

IPv6 does not use ARP.

Instead:

```text
IPv6
→ NDP / ICMPv6
```

**NDP = Neighbor Discovery Protocol**

Mental shortcut:

```text
IPv4 → ARP
IPv6 → NDP / ICMPv6
```

## IPv6 and NAT

Typical IPv4 home network:

```text
private IPv4 devices
      ↓
     NAT
      ↓
public IPv4
```

IPv6 has enough address space that NAT is not required simply to conserve addresses.

```text
IPv6
→ address-conservation NAT normally unnecessary
```

But:

```text
No NAT
≠
No firewall
```

Firewalling remains important.

---

# Stateful Firewalls

## NAT vs firewall

```text
NAT
→ translates addresses/ports
```

```text
Firewall
→ decides whether traffic is allowed
```

They are separate concepts.

## Stateful firewall

A stateful firewall remembers connection/flow state.

Example:

```text
Client
192.168.1.20:53124
        ↓
Server
203.0.113.10:443
```

The firewall can remember:

```text
This outbound connection exists.
```

Then return traffic can be recognized as belonging to an existing allowed flow.

```text
Stateful firewall
→ remembers connection state
```

---

# Connection Tracking

## Conntrack

**Connection tracking** keeps state about network flows.

Often called:

```text
conntrack
```

Conceptual entry:

```text
Protocol: TCP

10.0.0.5:53124
↔
203.0.113.10:443

State:
ESTABLISHED
```

Common high-level state ideas:

```text
NEW
ESTABLISHED
RELATED
INVALID
```

Exact names depend on implementation.

## Why conntrack matters

A firewall may allow outbound connections while blocking unsolicited inbound connections.

Return packets are technically incoming, but conntrack lets the firewall identify them as part of an already permitted flow.

---

# NAT State

NAT also commonly needs state.

Example:

```text
Private:
192.168.1.20:53124

        ↓ NAT

Public:
198.51.100.8:40001
```

The NAT device needs a mapping:

```text
198.51.100.8:40001
↔
192.168.1.20:53124
```

so returning traffic can be sent to the correct private endpoint.

## Unsolicited inbound traffic

Suppose:

```text
Internal service:
192.168.1.20:9000
```

Router public address:

```text
198.51.100.8
```

If someone sends traffic to:

```text
198.51.100.8:9000
```

the NAT router needs to know which internal machine should receive it.

Without an existing NAT mapping or port-forwarding rule, it normally cannot map that traffic to the private service.

## Port forwarding

Port forwarding creates an explicit mapping.

```text
198.51.100.8:9000
        ↓
192.168.1.20:9000
```

This may not be possible behind CGNAT because the local router may not control the public IPv4 address.

---

# NAT Traversal

## Concept

**NAT traversal** means techniques that help endpoints communicate through NAT/firewall boundaries.

Problem:

```text
Peer A
private network
   ↓ NAT

Internet

   NAT ↓
Peer B
private network
```

Direct communication may be difficult because neither side accepts arbitrary inbound traffic.

## NAT mappings from outbound traffic

An outbound packet can create a temporary NAT mapping.

Example:

```text
10.0.0.20:50000
      ↓ NAT
198.51.100.10:62000
```

Some NAT traversal techniques use these mappings to make peer-to-peer connectivity possible.

---

# STUN

**STUN** helps a client discover how it appears from outside its NAT.

Example:

```text
Client:
10.0.0.20:50000

   ↓ NAT

Public mapping:
198.51.100.10:62000

   ↓

STUN server
```

Mental shortcut:

```text
STUN
→ discover public-facing IP/port mapping
```

---

# Hole Punching

Very simplified UDP hole punching:

```text
Peer A → sends outward
Peer B → sends outward
```

Both NAT devices create temporary mappings.

Then direct UDP communication may become possible:

```text
Peer A ←──── UDP ────→ Peer B
```

This does not work with every NAT/firewall configuration.

---

# TURN / Relay

If direct peer-to-peer communication fails, a relay can forward traffic.

```text
Peer A
   ↓
Relay Server
   ↓
Peer B
```

Mental model:

```text
STUN
→ discover connectivity/public mapping

Hole punching
→ attempt direct peer connection

TURN
→ relay traffic when direct connection fails
```

---

# Why VPNs Care About NAT Traversal

VPN peers may both be behind NAT and firewalls, so direct peer communication can be difficult.

Real VPN systems may therefore use:

```text
persistent keepalives
NAT traversal
endpoint discovery
relay servers
```

---

# Linux Network Namespaces

A **network namespace** gives processes an isolated networking environment.

Conceptually:

```text
Linux Host
│
├─ Namespace A
│   ├─ interfaces
│   ├─ routes
│   └─ network/firewall state
│
└─ Namespace B
    ├─ interfaces
    ├─ routes
    └─ network/firewall state
```

Mental model:

```text
Network namespace
→ isolated virtual networking world
```

Containers commonly rely on this concept.

---

# Virtual Ethernet Pairs

Linux can create:

```text
veth pairs
```

Think of them as a virtual Ethernet cable:

```text
veth-A <==============> veth-B
```

Traffic entering one side appears on the other.

Common use:

```text
container / namespace
        ↕
      veth
        ↕
host networking
```

---

# One-Minute Cheat Sheet

```text
IPv6
→ 128-bit addressing
→ huge address space
```

```text
IPv6 neighbor discovery
→ NDP / ICMPv6
→ not ARP
```

```text
NAT
→ translates addresses/ports
```

```text
Firewall
→ permits or blocks traffic
```

```text
Stateful firewall
→ remembers connection/flow state
```

```text
Conntrack
→ tracks network flows/connections
```

```text
Port forwarding
→ explicit public-port → private-endpoint mapping
```

```text
NAT traversal
→ techniques for communicating through NAT/firewall boundaries
```

```text
STUN
→ discover public-facing IP/port mapping
```

```text
Hole punching
→ try direct peer-to-peer communication through NAT mappings
```

```text
TURN / Relay
→ forward traffic when direct communication fails
```

```text
Network namespace
→ isolated virtual networking environment
```

```text
veth pair
→ virtual Ethernet link between networking environments
```
