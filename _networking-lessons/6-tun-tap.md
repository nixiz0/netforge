# 6 — TUN / TAP

> Concise revision sheet for virtual network interfaces, TUN/TAP, routing into virtual interfaces, and userspace VPN packet flow.

---

# Virtual Network Interfaces

A virtual network interface is a software-created network interface.

Examples:

```text
eth0   → physical Ethernet
wlan0  → physical Wi-Fi
tun0   → virtual Layer-3 interface
tap0   → virtual Layer-2 interface
```

From the OS point of view, virtual interfaces participate in normal networking and routing.

---

# TUN vs TAP

```text
TUN
→ Layer 3
→ IP packets
```

```text
TAP
→ Layer 2
→ Ethernet frames
```

Mental shortcut:

```text
TUN → IP
TAP → Ethernet
```

---

# TUN Interface

A TUN interface gives userspace access to IP packets.

Example route:

```text
10.20.0.0/24 → tun0
```

Then traffic for that network follows:

```text
Application
   ↓
TCP / UDP
   ↓
IP packet
   ↓
Routing table
   ↓
tun0
   ↓
Userspace program
```

The userspace program can read the raw IP packet.

---

# VPN Send Path

Typical VPN flow:

```text
Application
   ↓
OS creates IP packet
   ↓
Routing chooses tun0
   ↓
VPN program reads packet from TUN
   ↓
Encrypt / authenticate
   ↓
Encapsulate
   ↓
Send through UDP/TCP socket
   ↓
Physical interface
   ↓
Internet
```

The packet read from TUN is the **inner packet**.

The VPN creates an **outer packet** to carry it across the real network.

---

# VPN Receive Path

Reverse direction:

```text
Internet
   ↓
VPN transport socket
   ↓
VPN program
   ↓
Authenticate / decrypt
   ↓
Decapsulate
   ↓
Recover inner IP packet
   ↓
Write packet to tun0
   ↓
Kernel network stack
   ↓
Local application
```

---

# Routing Controls TUN Traffic

Creating `tun0` alone does not make traffic use it.

The routing table decides what enters the TUN interface.

Example:

```text
10.20.0.0/24 → tun0
```

means:

```text
traffic for 10.20.0.x
→ send into tun0
```

So:

```text
TUN
→ receives IP packets

Routing table
→ decides which IP packets go there
```

---

# Full-Tunnel Routing

A full-tunnel VPN may route nearly all IPv4 traffic into TUN:

```text
0.0.0.0/0 → tun0
```

But the VPN server itself must still remain reachable through the physical network.

Example:

```text
VPN server:
203.0.113.20
```

A more specific route may be needed:

```text
203.0.113.20
→ physical gateway
```

while:

```text
0.0.0.0/0
→ tun0
```

Otherwise traffic needed to maintain the VPN tunnel could be routed back into the tunnel itself.

---

# TAP Interface

TAP gives userspace full Ethernet frames.

Example:

```text
[ Ethernet header | IP packet | trailer ]
```

This includes Layer-2 information such as:

```text
source MAC
destination MAC
EtherType
```

TAP can support:

```text
Ethernet broadcasts
ARP
bridging
non-IP Ethernet traffic
```

---

# Why Routed VPNs Often Use TUN

If a VPN only needs to route IP traffic, TUN is usually simpler.

```text
TUN
→ IP packets only
→ less Layer-2 overhead
→ natural for routed VPNs
```

TAP is useful when Layer-2 Ethernet behavior is specifically required.

---

# TUN Is Not the VPN

Important:

```text
TUN ≠ complete VPN
```

TUN only gives a userspace program access to IP packets.

A VPN still needs to implement:

```text
encapsulation
transport
encryption/authentication
peer communication
routing logic
key management
```

---

# Kernel Space vs Userspace

The OS network stack primarily operates in kernel space.

A VPN process usually runs in userspace.

TUN bridges them:

```text
Kernel routing
   ↓
TUN
   ↓
Userspace VPN process
```

and in reverse:

```text
Userspace VPN
   ↓
TUN
   ↓
Kernel network stack
```

---

# TUN IP Addresses

A TUN interface may have its own virtual IP.

Example:

```text
Client tun0:
10.8.0.2/24
```

Remote peer:

```text
Server tun0:
10.8.0.1
```

These are VPN/inner addresses.

A machine may simultaneously have:

```text
tun0       → 10.8.0.2
wlan0      → 192.168.1.50
public NAT → 198.51.100.10
```

These belong to different networking contexts.

---

# Packet Parsing

Reading from TUN provides bytes representing an IP packet.

Conceptually:

```text
byte[]
 ↓
IP header
 ↓
source IP
destination IP
protocol
payload
```

A VPN often does not need to understand the application protocol inside the packet.

It can often treat the packet mostly as opaque bytes:

```text
read
↓
encrypt
↓
encapsulate
↓
send
```

---

# Why TUN Is Powerful

Applications do not need to be explicitly configured to use a proxy.

Instead:

```text
Application
→ normal networking
→ OS routing
→ tun0
→ VPN
```

This allows many protocols to use the VPN transparently:

```text
HTTP/HTTPS
SSH
DNS
games
custom TCP/UDP protocols
```

---

# Permissions

Creating/configuring virtual interfaces and routes usually requires elevated networking privileges.

On Linux you may encounter:

```text
/dev/net/tun
root
CAP_NET_ADMIN
```

These are implementation details to learn during practice.

---

# One-Minute Cheat Sheet

```text
TUN
→ virtual Layer-3 interface
→ userspace receives IP packets
```

```text
TAP
→ virtual Layer-2 interface
→ userspace receives Ethernet frames
```

```text
Routing table
→ decides which traffic enters TUN
```

```text
Send path:
App → IP → route → TUN → VPN → encrypted tunnel → Internet
```

```text
Receive path:
Internet → VPN → decrypt → TUN → OS → App
```

```text
TUN ≠ VPN
```

```text
TUN
→ building block that connects OS IP routing to a userspace VPN
```

```text
Full tunnel
→ most traffic routed into TUN
```

```text
Specific route to VPN server
→ prevents tunnel traffic from recursively entering itself
```
