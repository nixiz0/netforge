# Networking Foundations — Revision Sheet

> **Module 1 · Lessons 1.1–1.14**  
> A compact revision sheet covering the essential foundations of computer networking.

---

# 1. Core ideas

## Network
A **network** is a group of devices that can exchange data.

```text
Laptop ─┐
Phone  ─┼─ LAN ─ Router ─ Internet
PC     ─┘
```

The Internet is a **network of networks**.

---

## IP address
Identifies a network interface at the IP layer.

```text
192.168.1.50
```

A computer can have several IPs:

```text
Laptop
├─ Wi-Fi    192.168.1.50
├─ Ethernet 10.0.0.20
└─ VPN/TUN  10.10.0.2
```

> Do not think: `1 computer = 1 IP`.

---

## Port
Identifies a transport endpoint used by an application.

```text
192.168.1.50:9000
                 └─ port
```

Useful model:

```text
IP   = where
Port = which network endpoint/application
```

A port is **not** the server itself. A server can listen on a port.

---

## Protocol
A set of communication rules.

Examples:

```text
Ethernet · IP · TCP · UDP · HTTP · ARP · DHCP
```

---

## Client / Server

```text
Client ── request ──> Server
Client <─ response ── Server
```

A proxy can be both:

```text
Browser ──> Proxy ──> Website

Browser → Proxy  : Proxy = server
Proxy → Website  : Proxy = client
```

---

# 2. LAN, Switch, Router

## LAN
**LAN = Local Area Network**

Devices on the same directly connected IP network can usually communicate without an IP router forwarding the packet.

```text
        Switch / Wi-Fi
       /      |       \
     PC     Laptop    Phone
```

## Switch
Mainly forwards Ethernet frames inside a LAN using:

```text
MAC addresses
```

Question it answers:

```text
Which port leads to this MAC?
```

## Router
Connects IP networks and forwards IP packets using:

```text
destination IP + routing table
```

Question it answers:

```text
Where should this packet go next?
```

A home "router" box often contains:

```text
Router + Switch + Wi-Fi AP + DHCP + NAT + Firewall
```

---

# 3. Encapsulation

Data is wrapped by protocols.

```text
Application data
      ↓
Transport
      ↓
IP packet
      ↓
Ethernet frame
```

Receiving does the reverse: **decapsulation**.

## Common names

```text
Ethernet → frame
IP       → packet
TCP      → segment
UDP      → datagram
```

## Headers

```text
Ethernet header → source/destination MAC
IP header       → source/destination IP
TCP header      → source/destination ports + TCP info
```

Visual:

```text
Ethernet Frame
┌─────────────────────────┐
│ src MAC / dst MAC       │
├─────────────────────────┤
│ IP Packet               │
│ ┌─────────────────────┐ │
│ │ src IP / dst IP     │ │
│ │ data...             │ │
│ └─────────────────────┘ │
└─────────────────────────┘
```

---

# 4. Network interfaces and MAC

A **network interface** is a networking endpoint exposed by the OS.

Examples:

```text
Wi-Fi
Ethernet
Loopback
VPN/TUN
```

A MAC address is a link-layer address associated with an Ethernet/Wi-Fi interface.

```text
AA:BB:CC:12:34:56
```

Typical Ethernet MAC:

```text
6 bytes = 48 bits
```

A computer can have several MAC addresses.

> A MAC is not the permanent identity of the whole computer. It can also be changed/randomized.

## MAC vs IP

```text
MAC → local-link delivery
IP  → logical communication across IP networks
```

For remote traffic:

```text
Ethernet dst MAC = next-hop router
IP dst           = final remote server
```

---

# 5. Loopback

```text
127.0.0.1
```

is the common IPv4 loopback address.

```text
localhost = this machine
```

Example:

```text
127.0.0.1:9000
```

can test a local server without leaving the machine.

---

# 6. How switches learn

A switch learns from the **source MAC**.

```text
Frame arrives on port 3

src MAC = AA
dst MAC = BB

Switch learns:
AA → port 3
```

Then it forwards using the **destination MAC**.

```text
LEARN   = source MAC
FORWARD = destination MAC
```

If destination is known:

```text
BB → port 7
```

→ forward only to port 7.

If unknown:

```text
switch floods the frame
```

It does **not** actively ask where BB is.

---

## Ethernet broadcast

```text
FF:FF:FF:FF:FF:FF
```

means:

```text
everyone in the local Layer-2 broadcast domain
```

Routers normally do not forward ordinary Ethernet broadcasts between networks.

---

# 7. ARP

**ARP = Address Resolution Protocol**

Purpose:

```text
I know the local/next-hop IPv4 address.
What MAC should I use?
```

Mapping:

```text
IP → MAC
```

Example:

```text
192.168.1.20 → BB:BB:BB:BB:BB:BB
```

If unknown, ARP broadcasts:

```text
Who has 192.168.1.20?
```

The owner replies with its MAC.

## ARP cache

```text
192.168.1.1  → Router MAC
192.168.1.20 → PC B MAC
```

## Do not confuse

```text
ARP cache        : IP  → MAC
Switch MAC table : MAC → switch port
```

Together:

```text
IP
↓
ARP
↓
MAC
↓
Switch table
↓
switch port
```

---

## Remote destination

For:

```text
8.8.8.8
```

your PC normally does **not** ARP for `8.8.8.8`.

It does:

```text
destination 8.8.8.8
↓
routing decision
↓
next hop = gateway 192.168.1.1
↓
ARP 192.168.1.1
↓
gateway MAC
```

Each router resolves **its own next hop**.

---

# 8. DHCP

**DHCP = Dynamic Host Configuration Protocol**

It can automatically provide:

```text
IP address
subnet/prefix
default gateway
DNS server
lease duration
```

## DORA

```text
Discover
Offer
Request
Acknowledge
```

```text
Client                    DHCP Server

DISCOVER ------------------>
         <------------- OFFER
REQUEST ------------------->
         <--------------- ACK
```

Broadcasting is useful because the new client may not yet know:

```text
its usable IP
DHCP server address
```

## Lease
DHCP assignments are usually temporary so addresses can be reused.

Example pool:

```text
192.168.1.100 - 192.168.1.200
```

---

# 9. CIDR and subnet masks

IPv4:

```text
32 bits
```

Example:

```text
192.168.1.50/24
```

`/24` means:

```text
first 24 bits = network prefix
last 8 bits   = host part
```

Equivalent mask:

```text
255.255.255.0
```

Visual:

```text
192 . 168 . 1 | 50
──────────────  ──
 network        host
```

Network:

```text
192.168.1.0/24
```

Typical:

```text
Network   : 192.168.1.0
Hosts     : 192.168.1.1 - 192.168.1.254
Broadcast : 192.168.1.255
```

---

## Why the prefix matters

```text
192.168.1.10/24
192.168.2.10/24
```

→ different `/24` networks.

But:

```text
192.168.1.10/16
192.168.2.10/16
```

→ both belong to:

```text
192.168.0.0/16
```

So:

> You cannot decide local/remote from the IP alone.

You need the **prefix and routing table**.

---

## Useful masks

```text
/8  = 255.0.0.0
/16 = 255.255.0.0
/24 = 255.255.255.0
/25 = 255.255.255.128
/26 = 255.255.255.192
/27 = 255.255.255.224
```

## Addresses per subnet

```text
2^(32 - prefix)
```

```text
/24 → 256
/25 → 128
/26 → 64
/27 → 32
```

Usual IPv4 subnet:

```text
usable hosts ≈ total - network - broadcast
```

---

## Example `/26`

```text
192.168.1.70/26
```

`/26` → blocks of 64:

```text
0-63
64-127
128-191
192-255
```

`70` is inside `64-127`.

```text
Network   : 192.168.1.64
Hosts     : 192.168.1.65 - 192.168.1.126
Broadcast : 192.168.1.127
```

---

## Special prefixes

```text
0.0.0.0/0
```

matches every IPv4 address → commonly the default route.

```text
x.x.x.x/32
```

matches exactly one IPv4 address.

---

# 10. Routing table

Purpose:

```text
For this destination IP,
where should the packet go?
```

A route can contain:

```text
destination prefix
outgoing interface
gateway / next hop
metric
```

Example:

```text
192.168.1.0/24 → directly via Wi-Fi
0.0.0.0/0      → gateway 192.168.1.1
```

## Direct route

```text
192.168.1.80
↓
matches 192.168.1.0/24
↓
ARP 192.168.1.80
↓
send directly
```

## Gateway route

```text
8.8.8.8
↓
default route
↓
gateway 192.168.1.1
↓
ARP gateway
↓
send frame to router
```

---

## Longest prefix match

Routes:

```text
10.0.0.0/8      → A
10.20.0.0/16    → B
10.20.30.0/24   → C
0.0.0.0/0       → D
```

Destination:

```text
10.20.30.50
```

All match, but `/24` is most specific.

```text
→ choose C
```

Rule:

```text
most specific matching prefix wins
```

---

## Hop by hop

```text
PC → Router A → Router B → Router C → Server
```

Each device chooses only its **next hop**.

```text
PC       → A
Router A → B
Router B → C
```

---

# 11. Private and public IPv4

Private ranges:

```text
10.0.0.0/8
172.16.0.0/12
192.168.0.0/16
```

Private IPs are not globally routed on the normal public Internet.

Two homes can both use:

```text
192.168.1.50
```

because they are separate private networks.

A public IPv4 address is globally routable.

Typical home:

```text
Laptop
192.168.1.50
      ↓
Router
LAN: private
WAN: public
      ↓
Internet
```

> One home does not necessarily equal one public IPv4. Multiple public IPs or CGNAT are possible.

---

# 12. NAT

**NAT = Network Address Translation**

Common home NAT:

```text
Before:
src = 192.168.1.10

After:
src = router public IP
```

Ports are often translated too.

Example NAT state:

```text
Public
203.0.113.5:40001

      ↕

Private
192.168.1.10:51000
```

This lets several private devices share one public IPv4.

---

## Routing vs ARP vs NAT vs Firewall

```text
Routing
→ Where should the packet go?

ARP
→ What local MAC corresponds to the IPv4 next hop?

NAT
→ Should addresses/ports be translated?

Firewall
→ Is the traffic allowed?
```

---

# 13. Port forwarding

For an internal server:

```text
192.168.1.50:9000
```

an inbound rule can map:

```text
public-IP:9000
      ↓
192.168.1.50:9000
```

This allows externally initiated traffic to reach that internal service.

It also increases attack surface, so the service should have appropriate authentication, patching and firewall/access rules.

---

# 14. CGNAT

Sometimes the ISP also performs NAT:

```text
Laptop
private IP
   ↓
Home NAT
   ↓
ISP shared address
   ↓
CGNAT
   ↓
Internet
```

This can make inbound hosting harder because you do not control the ISP NAT.

---

# 15. Complete flow

Example:

```text
PC      : 192.168.1.50/24
Gateway : 192.168.1.1
Server  : remote IP
```

```text
Application wants remote IP
          ↓
Routing table lookup
          ↓
Choose route + next hop
          ↓
ARP next-hop IPv4 → MAC
          ↓
Build Ethernet frame
          ↓
Switch / local link
          ↓
Router
          ↓
Possible NAT
          ↓
Next router
          ↓
...
          ↓
Server
```

First local frame:

```text
Ethernet
src MAC = PC
dst MAC = Router

IP
src IP = PC
dst IP = Server
```

The MAC addresses can change hop by hop.

The IP packet remains aimed at the final IP destination unless something such as NAT modifies it.

---

# 16. Fast decision model

```text
Destination IP
      ↓
Routing table
      ↓
Most specific route
      ↓
┌─────────────────────┬─────────────────────┐
│ Directly connected  │ Via gateway         │
├─────────────────────┼─────────────────────┤
│ ARP destination IP  │ ARP next-hop IP     │
│ get host MAC        │ get gateway MAC     │
│ send directly       │ send to router      │
└─────────────────────┴─────────────────────┘
```

---

# 17. Must-remember mappings

```text
ARP cache
IP → MAC
```

```text
Switch table
MAC → switch port
```

```text
Routing table
destination prefix → next hop / interface
```

```text
NAT table
public translated endpoint ↔ private endpoint
```

```text
DHCP
device → network configuration
```

```text
DNS
domain name → records such as A/AAAA
```

```text
ICMP
diagnostic/control/error information for IP
```

---

# 18. DNS

**DNS = Domain Name System**

DNS lets humans use names instead of remembering destination IP addresses.

```text
example.com
    ↓
DNS
    ↓
IP address
```

DNS does **not** route packets.

```text
DNS     → obtain destination information such as an IP
Routing → decide where the IP packet goes
```

## Important record types

```text
A     → IPv4 address
AAAA  → IPv6 address
```

DNS can also contain records such as `CNAME`, `MX`, `NS`, and `TXT`.

## Recursive vs authoritative

```text
Recursive resolver
→ finds an answer for the client
```

```text
Authoritative DNS server
→ holds authoritative records for a DNS zone/domain
```

Simplified lookup when nothing is cached:

```text
Client
  ↓
Recursive resolver
  ↓
Root
  ↓
TLD (.com, .fr, ...)
  ↓
Authoritative DNS server
  ↓
Answer
```

The root does not normally store the final address for every domain; DNS uses delegation.

## DNS cache and TTL

DNS answers are cached for efficiency.

Possible caches:

```text
Browser
OS
router/local resolver
ISP/public recursive resolver
```

DNS **TTL** tells caches roughly how long a record may be reused before it expires.

```text
DNS TTL
→ cache lifetime, usually expressed in seconds
```

## DNS in the full flow

```text
Domain name
    ↓
DNS
    ↓
Destination IP
    ↓
Routing table
    ↓
Next hop
    ↓
ARP if needed
    ↓
Ethernet frame
```

---

# 19. ICMP, ping, traceroute and IP TTL

**ICMP = Internet Control Message Protocol**

ICMP carries network control, diagnostic and error information.

Examples:

```text
Echo Request / Echo Reply
Time Exceeded
Destination Unreachable
```

ICMP is carried directly in IP; normal ICMP does **not** use TCP/UDP ports.

## ping

`ping` commonly uses:

```text
ICMP Echo Request
        ↓
destination

ICMP Echo Reply
        ↑
```

A successful ping shows ICMP reachability and gives a round-trip time (RTT).

A failed ping does **not** automatically mean the server is offline.

Possible reasons:

```text
ICMP filtered
firewall rules
packet loss
routing problem
destination does not answer Echo
```

A service such as HTTPS may still work.

## IPv4 TTL

The IPv4 header contains a **TTL — Time To Live**.

Routers decrement it when forwarding:

```text
TTL 4
 ↓ Router A
TTL 3
 ↓ Router B
TTL 2
 ↓ Router C
TTL 1
```

When a router decrements TTL to zero, it discards the packet and may return:

```text
ICMP Time Exceeded
```

TTL prevents packets from circulating forever in routing loops.

## traceroute

Traceroute discovers responding hops using probes with increasing TTL values:

```text
Probe 1: TTL 1 → Router A expires it
Probe 2: TTL 2 → Router B expires it
Probe 3: TTL 3 → Router C expires it
...
```

Each responding router can reveal another hop.

Traceroute is useful but not a perfect physical map: routers may filter probes, paths can change, and load balancing can affect results.

## Do not confuse the two TTLs

```text
DNS TTL
→ how long DNS data may remain cached
```

```text
IP TTL
→ hop limit for an IPv4 packet
```

---

# 20. MTU, fragmentation and MSS

**MTU = Maximum Transmission Unit**

MTU is the maximum IP packet size that a particular link/interface can carry without requiring fragmentation for that link.

A common Ethernet MTU is:

```text
1500 bytes
```

The application does not get all 1500 bytes because protocol headers consume space.

Simplified IPv4/TCP example:

```text
MTU             1500
- IPv4 header     20
- TCP header      20
────────────────────
TCP payload     1460
```

Actual header sizes can vary.

## Fragmentation

If an IPv4 packet is too large for a link, it may need to be fragmented in situations where fragmentation is permitted:

```text
Large IP packet
      ↓
Fragment 1
Fragment 2
Fragment 3
```

The destination reassembles the original packet.

Fragmentation is generally undesirable because it adds overhead and complexity, and losing a fragment prevents complete reassembly of that original packet.

## Interface MTU vs Path MTU

```text
Interface MTU
→ maximum supported/configured packet size on one interface/link
```

```text
Path MTU
→ largest packet that can cross the complete path without fragmentation
```

The Path MTU is constrained by the smallest relevant MTU along the route.

```text
PC ─1500─ Router ─1400─ Router ─1500─ Server
                    ↑
             Path MTU ≈ 1400
```

## Path MTU Discovery

Hosts can try to discover a usable packet size for the path rather than relying on fragmentation.

ICMP can be important in this process, which is one reason indiscriminately blocking all ICMP can break network behavior.

## MTU vs TCP MSS

```text
MTU
→ maximum IP packet size for the link
```

```text
TCP MSS
→ maximum TCP payload size a peer advertises
```

In a simple IPv4/TCP case:

```text
1500 MTU - 20 IP - 20 TCP ≈ 1460 MSS
```

## Why VPNs care about MTU

A tunnel adds extra bytes:

```text
Outer IP / transport
        ↓
VPN/tunnel headers
        ↓
encryption overhead
        ↓
original IP packet
```

If the original packet already fills the physical MTU, the wrapped packet becomes too large.

Therefore a VPN/TUN interface often uses a smaller MTU, for example:

```text
Physical interface MTU = 1500
VPN/TUN MTU            = 1400
```

The exact value depends on tunnel overhead and the network path.

---

# 21. One-minute cheat sheet

```text
IP       = logical network address
Port     = transport/application endpoint
MAC      = local link-layer address
Switch   = forwards frames using MAC
Router   = forwards packets using routes
ARP      = next-hop IPv4 → MAC
DHCP     = automatic network configuration
DNS      = domain name → DNS records / IPs
CIDR     = network prefix length
Routing  = chooses next hop/interface
ICMP     = IP control, diagnostic and error messages
NAT      = translates addresses/ports
Firewall = permits/blocks traffic
MTU      = maximum IP packet size for a link
MSS      = maximum advertised TCP payload size
```

```text
Directly connected:
destination IP → ARP destination → MAC → frame

Remote:
destination IP → route → gateway → ARP gateway → frame
```

```text
/24 = 255.255.255.0   = 256 addresses
/25 = 255.255.255.128 = 128
/26 = 255.255.255.192 = 64
/27 = 255.255.255.224 = 32
```

---
