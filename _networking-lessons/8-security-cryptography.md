# 8 — Security & Cryptography

> Concise revision sheet for confidentiality, integrity, authentication, symmetric/asymmetric cryptography, hashes, HMAC/MAC, AEAD, nonces, key exchange, forward secrecy, replay protection, and VPN packet security.

---

# Security Goals

A secure VPN needs to answer:

```text
Can someone read my traffic?
Can someone modify my traffic?
Am I talking to the correct peer?
```

These map to:

```text
Confidentiality
Integrity
Authentication
```

---

# Confidentiality

Confidentiality means unauthorized parties should not be able to understand the data.

```text
plaintext
   ↓
encryption
   ↓
ciphertext
```

Mental model:

```text
Confidentiality
→ protect data from being read
```

---

# Integrity

Integrity means unauthorized modification should be detectable.

Example:

```text
amount=100
```

should not be silently changed to:

```text
amount=900
```

Mental model:

```text
Integrity
→ detect modification
```

---

# Authentication

Authentication verifies who you are communicating with.

For VPN peers:

```text
Peer A
   ↓
"Is this really Peer B?"
```

Mental model:

```text
Authentication
→ verify identity/origin
```

A secure VPN normally needs all three:

```text
confidentiality
+
integrity
+
authentication
```

---

# Symmetric Cryptography

Symmetric cryptography uses shared secret key material.

```text
shared secret key
     ↓
encrypt / decrypt
```

Advantages:

```text
fast
efficient
good for large amounts of traffic
```

This is why VPN packet encryption normally relies heavily on symmetric cryptography.

---

# Asymmetric Cryptography

Asymmetric cryptography uses a key pair:

```text
Public key
Private key
```

```text
Public key
→ may be shared

Private key
→ must remain secret
```

Useful for:

```text
authentication
digital signatures
key establishment
```

It is generally more expensive than symmetric cryptography.

---

# Combining Symmetric and Asymmetric Cryptography

A common design is:

```text
Asymmetric cryptography
        ↓
authenticate / establish shared secret
        ↓
Symmetric session keys
        ↓
encrypt normal traffic efficiently
```

---

# Hash Functions

A cryptographic hash takes arbitrary input and produces a fixed-size output.

```text
"hello"
   ↓
HASH
   ↓
fixed-size digest
```

Important ideas:

```text
same input
→ same hash

small input change
→ very different hash

hash
→ designed to be one-way
```

---

# Hashing vs Encryption

```text
Encryption
→ reversible with the appropriate key
```

```text
Hashing
→ designed to be one-way
```

You do not normally "decrypt a hash."

Hashes are useful for:

```text
fingerprints
integrity-related constructions
password systems
digital signatures
```

A plain hash alone does not authenticate a packet against an active attacker.

---

# MAC — Message Authentication Code

In cryptography:

```text
MAC
= Message Authentication Code
```

This is unrelated to a network MAC address.

A MAC uses secret key material to authenticate data.

```text
message
+
secret key
   ↓
MAC algorithm
   ↓
authentication tag
```

Receiver verifies using the corresponding secret.

A cryptographic MAC provides:

```text
integrity
+
message authentication
```

---

# HMAC

**HMAC** is a MAC construction based on a cryptographic hash plus a secret key.

```text
secret key
+
message
↓
HMAC
↓
authentication tag
```

Mental shortcut:

```text
Hash
→ no shared secret by itself

HMAC
→ hash-based authentication using a secret
```

---

# AEAD

**AEAD = Authenticated Encryption with Associated Data**

AEAD combines:

```text
encryption
+
integrity/authentication
```

Conceptually:

```text
plaintext
+
key
+
nonce
   ↓
AEAD encrypt
   ↓
ciphertext + authentication tag
```

Receiver:

```text
ciphertext
+
tag
+
key
+
nonce
   ↓
AEAD decrypt
   ↓
plaintext OR failure
```

If ciphertext or authenticated data is modified:

```text
verification fails
```

AEAD is highly useful for VPN packet protection.

---

# Associated Data

Associated Data is:

```text
authenticated
but not encrypted
```

Example conceptual packet:

```text
visible header
→ authenticated

payload
→ encrypted + authenticated
```

This lets a protocol keep some metadata visible while still protecting it from modification.

---

# Nonces

A nonce is a value used according to the cryptographic algorithm's uniqueness requirements.

Example:

```text
packet 1 → nonce 1
packet 2 → nonce 2
packet 3 → nonce 3
```

Important:

```text
Nonce reuse with the same key
→ can be catastrophic for many AEAD schemes
```

A nonce usually does **not** need to be secret.

The important property is correct use, especially avoiding dangerous reuse.

---

# Key Exchange

Symmetric encryption requires peers to establish shared secret key material.

Key exchange solves:

```text
How do two peers establish a shared secret
over an untrusted network?
```

Conceptually:

```text
Peer A                 Peer B

private info           private info
public info ---------- public info

      ↓ cryptographic exchange ↓

Both derive:
shared secret
```

---

# Diffie–Hellman Concept

Diffie–Hellman is a famous family of key-exchange techniques.

Modern protocols often use elliptic-curve variants.

High-level idea:

```text
A has private secret
B has private secret

exchange public values

A derives shared secret
B derives same shared secret
```

The final shared secret does not need to be sent directly across the network.

---

# Key Exchange Is Not Authentication

Key exchange answers:

```text
Can we establish shared secret material?
```

Authentication answers:

```text
Who did I establish it with?
```

Secure protocols combine:

```text
key exchange
+
authentication
```

Otherwise a man-in-the-middle may be possible.

---

# Digital Signatures

A digital signature uses asymmetric cryptography.

Conceptually:

```text
Private key
+
message
↓
signature
```

The corresponding public key can verify the signature.

Digital signatures help provide:

```text
authenticity
integrity
```

The private signing key must remain secret.

---

# Session Keys

Practical systems often derive temporary session keys.

```text
long-term identity
      ↓
authenticated key exchange
      ↓
temporary session keys
      ↓
protect packet traffic
```

Session keys may later be replaced.

---

# Forward Secrecy

Forward secrecy aims to ensure that a later compromise of long-term keys does not automatically expose previously recorded sessions.

Mental model:

```text
long-term key compromised later
≠
old sessions automatically decryptable
```

Ephemeral key-exchange secrets can help provide this property.

---

# Replay Attacks

An attacker may record a valid encrypted packet and send it again later.

```text
original:
Peer A → packet #100 → Peer B

later:
Attacker → packet #100 → Peer B
```

If accepted again, this is a replay attack.

---

# Replay Protection

A VPN protocol can track:

```text
sequence numbers
packet counters
nonces
replay windows
```

Example:

```text
received:
100
101
102

received again:
100
```

Receiver:

```text
already processed
→ reject
```

Mental model:

```text
Replay protection
→ reject previously valid packets sent again
```

---

# Rekeying

Keys should not necessarily remain in use forever.

A VPN may periodically derive fresh keys based on:

```text
time
traffic volume
session lifetime
security policy
```

This is called:

```text
rekeying
```

---

# Authentication vs Encryption

Do not confuse them.

```text
Encryption
→ hides data
```

```text
Authentication
→ verifies identity/origin
```

Encrypted communication can still be unsafe if you encrypted data to the wrong peer.

---

# VPN Packet Security

A conceptual secure VPN packet might look like:

```text
[ visible VPN header ]
[ encrypted inner IP packet ]
[ authentication tag ]
```

The protocol may track:

```text
peer
packet counter
nonce
session key
```

Receiving side:

```text
packet arrives
   ↓
identify peer/session
   ↓
check replay protection
   ↓
authenticate packet
   ↓
decrypt
   ↓
recover inner IP packet
   ↓
write to TUN
```

---

# Cryptography Engineering Rule

Important:

```text
Do not invent your own cipher.
```

And generally:

```text
Do not invent your own cryptographic protocol
without deep expertise and review.
```

Prefer:

```text
standard algorithms
well-reviewed constructions
well-reviewed libraries
```

Educational implementations can be useful for learning, but real security should rely on established designs.

---

# Full VPN Security Model

```text
Peer identity
   ↓
authentication
   ↓
authenticated key exchange
   ↓
session keys
   ↓
AEAD encryption
   ↓
nonce / packet counter
   ↓
replay protection
   ↓
secure tunnel packets
```

Send:

```text
TUN packet
   ↓
encrypt + authenticate
   ↓
tunnel transport
   ↓
Internet
```

Receive:

```text
tunnel packet
   ↓
authenticate
   ↓
replay check
   ↓
decrypt
   ↓
TUN packet
```

---

# One-Minute Cheat Sheet

```text
Confidentiality
→ prevent unauthorized reading
```

```text
Integrity
→ detect modification
```

```text
Authentication
→ verify identity/origin
```

```text
Symmetric crypto
→ fast encryption with shared secret keys
```

```text
Asymmetric crypto
→ public/private keys
→ authentication / signatures / key establishment
```

```text
Hash
→ one-way fixed-size digest
```

```text
HMAC / MAC
→ keyed message authentication
```

```text
AEAD
→ encryption + integrity/authentication
```

```text
Nonce
→ usually public
→ must obey uniqueness rules
```

```text
Key exchange
→ establish shared secret material
```

```text
Forward secrecy
→ later long-term key compromise should not expose old sessions
```

```text
Replay protection
→ reject old valid packets sent again
```

```text
Rekeying
→ periodically replace session keys
```

```text
VPN security
→ authenticated key exchange + session keys + AEAD + replay protection
```
