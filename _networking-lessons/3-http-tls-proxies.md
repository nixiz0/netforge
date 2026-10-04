# 3 — HTTP, TLS, and Proxies

> Concise revision sheet for HTTP, HTTPS/TLS, forward/reverse proxies, CONNECT, and SOCKS5.

---

# HTTP Essentials

## HTTP

**HTTP = Hypertext Transfer Protocol**

HTTP is an application-layer **request/response protocol**.

```text
Client                    Server

HTTP request ------------>

          <-------------- HTTP response
```

HTTP usually runs over TCP.

```text
HTTP  → TCP
HTTPS → TLS → TCP
```

HTTP/3 is different and runs over QUIC/UDP, but the request/response semantics remain HTTP.

---

## HTTP request structure

Simplified HTTP/1.1 request:

```http
GET /hello HTTP/1.1
Host: example.com
User-Agent: MyBrowser
Accept: text/html

```

Structure:

```text
Request line
↓
Headers
↓
Blank line
↓
Optional body
```

Example request line:

```text
GET /hello HTTP/1.1
↑   ↑       ↑
|   |       HTTP version
|   path
method
```

### Common methods

```text
GET     → retrieve a resource
POST    → send/create/trigger data
PUT     → replace/update a resource
PATCH   → partial update
DELETE  → delete a resource
HEAD    → GET-like request without a normal response body
```

---

## HTTP response structure

Example:

```http
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: 5

hello
```

Structure:

```text
Status line
↓
Headers
↓
Blank line
↓
Optional body
```

### Status code families

```text
1xx → informational
2xx → success
3xx → redirection
4xx → client/request problem
5xx → server-side problem
```

Useful examples:

```text
200 OK
201 Created

301 Moved Permanently
302 Found

400 Bad Request
401 Unauthorized
403 Forbidden
404 Not Found

500 Internal Server Error
502 Bad Gateway
503 Service Unavailable
```

---

## HTTP headers

Headers carry metadata about requests and responses.

Examples:

```http
Host: example.com
Content-Type: application/json
Content-Length: 123
Authorization: Bearer ...
Cookie: ...
```

Headers can describe:

```text
content type
content length
authentication
cookies
caching
compression
host
connection behavior
```

---

## Host header and virtual hosting

One IP can serve multiple websites.

```text
203.0.113.20
├─ example.com
├─ shop.example.com
└─ another-site.com
```

The HTTP request can contain:

```http
Host: example.com
```

This lets the server know which virtual host/site the client wants.

---

## URL breakdown

Example:

```text
https://example.com:443/users?id=42
```

```text
https
→ scheme

example.com
→ hostname

443
→ port

/users
→ path

?id=42
→ query string
```

Common defaults:

```text
HTTP  → port 80
HTTPS → port 443
```

---

## HTTP framing

TCP is only a byte stream.

```text
TCP
→ ordered bytes
→ no application message boundaries
```

Therefore HTTP needs its own framing rules.

One method:

```http
Content-Length: 100
```

which tells the receiver how many bytes belong to the body.

HTTP/1.1 can also use chunked transfer encoding.

Key idea:

```text
HTTP must define message boundaries
because TCP does not.
```

---

## Persistent connections

HTTP connections can be reused.

```text
TCP connection
   ↓
GET /index.html
← response

GET /style.css
← response

GET /image.png
← response
```

This avoids repeating connection setup.

Benefits:

```text
less TCP handshake overhead
less TLS handshake overhead for HTTPS
lower latency
```

---

## HTTP is stateless

At the protocol level, requests can be treated independently.

Applications add state using:

```text
cookies
session identifiers
tokens
databases
```

Example:

```http
Set-Cookie: session=abc123
```

Later:

```http
Cookie: session=abc123
```

---

# HTTPS and TLS

## HTTPS

HTTPS is HTTP protected by TLS.

```text
HTTP
 ↓
TLS
 ↓
TCP
 ↓
IP
```

TLS = **Transport Layer Security**.

Its three essential goals are:

```text
confidentiality
integrity
authentication
```

---

## Confidentiality

Confidentiality means unauthorized observers should not be able to read the application data.

```text
plaintext HTTP
→ readable if intercepted
```

```text
HTTPS/TLS
→ encrypted in transit
```

---

## Integrity

Integrity means traffic should not be silently modified without detection.

```text
transfer=100
```

should not be changeable into:

```text
transfer=10000
```

without the modification being detected.

Mental model:

```text
Confidentiality
→ cannot read

Integrity
→ cannot modify unnoticed
```

---

## Authentication

TLS also needs to verify the identity of the server.

Otherwise an attacker could sit between client and server:

```text
Client
  ↓
Attacker
  ↓
Real Server
```

Authentication helps the client verify:

```text
"I am really talking to the intended server."
```

---

## Certificates

A TLS server presents a certificate.

Conceptually it contains information such as:

```text
domain/identity information
public-key information
validity period
digital signatures
```

The client checks that the certificate is valid for the requested hostname and chains to a trusted authority.

---

## Certificate Authorities

**CA = Certificate Authority**

Simplified trust chain:

```text
Trusted Root CA
      ↓
Intermediate CA
      ↓
Website certificate
      ↓
example.com
```

The client does not blindly trust any certificate the server sends.

---

## Symmetric vs asymmetric cryptography

### Symmetric

Uses shared secret key material.

```text
shared secret
→ encrypt/decrypt
```

Useful because it is efficient for large amounts of traffic.

TLS uses symmetric cryptography for bulk application data after the handshake.

### Asymmetric

Uses a key pair:

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

Public-key cryptography helps with authentication and key establishment.

---

## TLS handshake

High-level model:

```text
Client                        Server

Hello ---------------------->

        <---- Server hello
              Certificate
              key exchange information

Client verifies certificate

Both derive shared session secrets

<------ secure channel established ------>

HTTP traffic
```

The exact TLS 1.3 handshake is more detailed, but the essential result is:

```text
server authenticated
+
shared session keys established
```

Then application data can be protected with efficient symmetric cryptography.

---

## What TLS protects

TLS protects:

```text
application data in transit
```

It does not hide all metadata.

Observers may still learn things such as:

```text
source IP
destination IP
traffic size
timing
```

And TLS does not protect data from the legitimate endpoint itself.

```text
Client ← TLS → Server
```

The server still receives the application data in usable form.

---

# Proxies

## Proxy

A proxy is an intermediary that receives traffic from one side and forwards it toward another destination.

Two major models:

```text
Forward proxy
→ represents the client side
```

```text
Reverse proxy
→ represents the server side
```

---

## Forward proxy

Normal path:

```text
Client --------------------> Website
```

With forward proxy:

```text
Client ----> Proxy ----> Website
```

Possible uses:

```text
access control
filtering
logging
caching
hiding the client's direct IP from the destination
routing traffic through another network
```

The proxy acts as:

```text
server toward the client
client toward the destination
```

---

## Reverse proxy

A reverse proxy sits in front of backend servers.

```text
Client ----> Reverse Proxy ----> Backend
```

It may route requests to multiple services:

```text
                ┌→ Backend A
Client → Proxy ─┼→ Backend B
                └→ Backend C
```

Common uses:

```text
load balancing
TLS termination
caching
security filtering
request routing
```

Mental shortcut:

```text
Forward proxy
→ represents clients

Reverse proxy
→ represents servers
```

---

## Plain HTTP proxying

A browser configured to use an HTTP proxy can send something like:

```http
GET http://example.com/users HTTP/1.1
Host: example.com
```

The proxy can determine the destination and create its own connection to the website.

```text
Browser
   ↓ HTTP
Proxy
   ↓ HTTP
Website
```

Because plain HTTP is not encrypted, the proxy can potentially inspect or modify the HTTP contents.

---

# HTTPS through an HTTP proxy

## HTTP CONNECT

For HTTPS, the browser normally wants TLS to remain between itself and the destination website.

It first asks the HTTP proxy:

```http
CONNECT example.com:443 HTTP/1.1
Host: example.com:443
```

Meaning roughly:

```text
"Open a TCP connection to example.com:443 for me."
```

The proxy connects to the destination and then forwards bytes between both sides.

```text
Browser
   ↓
Proxy
   ↓
Website
```

After CONNECT succeeds:

```text
Browser <====== TLS ======> Website
          through proxy
```

The proxy normally sees encrypted TLS bytes rather than plaintext HTTP.

Key idea:

```text
CONNECT
→ creates a TCP tunnel

TLS
→ happens through that tunnel
```

---

## What the CONNECT proxy can still know

Even without decrypting HTTPS, the proxy can know metadata such as:

```text
client connection
destination host/IP
destination port
connection timing
traffic volume
```

With CONNECT, the proxy normally sees the requested destination such as:

```text
example.com:443
```

But it does not automatically see encrypted content such as:

```text
GET /private/account
Cookie: ...
response body
```

---

## TLS interception

Advanced concept:

A proxy can deliberately terminate TLS and create another TLS connection toward the real server.

```text
Browser
  ⇅ TLS #1
Proxy
  ⇅ TLS #2
Website
```

Then the proxy can inspect HTTPS contents.

For this to work without certificate warnings, the client must trust the proxy's certificate authority.

This may be called:

```text
TLS interception
HTTPS inspection
MITM proxying
```

---

# SOCKS5

## SOCKS5 concept

SOCKS5 is a more generic proxy protocol.

Instead of understanding HTTP requests, the client can essentially ask:

```text
Connect me to:
example.com:443
```

Then:

```text
Client <---- bytes ----> SOCKS5 Proxy <---- bytes ----> Destination
```

SOCKS5 is not limited to HTTP.

It can proxy generic TCP connections such as:

```text
HTTPS
SSH
database connections
custom TCP protocols
```

SOCKS5 also supports UDP association; only the concept is important here.

---

## SOCKS5 handshake

Very simplified:

```text
Client → Proxy
```

1. Negotiate authentication method.
2. Request a connection to a host/port.
3. Proxy establishes the destination connection.
4. Data is relayed.

SOCKS5 is a binary protocol.

---

## DNS with SOCKS5

Two possible models:

```text
Client resolves domain
↓
Proxy receives destination IP
```

or:

```text
Client sends domain name
↓
Proxy resolves it
```

This can affect privacy and accessibility.

---

# HTTP Proxy vs SOCKS5

| HTTP Proxy | SOCKS5 |
|---|---|
| Understands HTTP semantics | Generic proxy protocol |
| Can inspect plain HTTP | Usually treats application data as bytes |
| HTTPS commonly uses CONNECT | Requests host:port directly |
| Mostly web-oriented | More protocol-agnostic |
| HTTP-style textual protocol | Binary protocol |

---

# Proxy vs VPN

A proxy normally works at the application/connection level.

```text
App A → Proxy
App B → normal network
App C → normal network
```

A VPN works at the IP/routing level and can carry broader system traffic.

```text
App A ─┐
App B ─┼→ routing → VPN tunnel
App C ─┘
```

So:

```text
Proxy
→ application/connection-level forwarding
```

```text
VPN
→ IP routing/tunneling
```

A proxy is not the same thing as a VPN.

---

# Full Mental Model

### Plain HTTP proxy

```text
Browser
  ↓ HTTP
Proxy
  ↓ HTTP
Website
```

### HTTPS via CONNECT

```text
Browser
  ↓ CONNECT
Proxy
  ↓ TCP connection
Website

then

Browser <====== encrypted TLS ======> Website
                  through Proxy
```

### SOCKS5

```text
Application
   ↓
SOCKS5:
"connect to host:port"
   ↓
Proxy
   ↓
Destination
```

### Reverse proxy

```text
Client
   ↓
Reverse Proxy
   ↓
Backend Server(s)
```

---

# One-Minute Cheat Sheet

```text
HTTP
→ request/response application protocol
```

```text
Request
→ method + path + headers + optional body
```

```text
Response
→ status + headers + optional body
```

```text
HTTPS
→ HTTP over TLS
```

```text
TLS
→ confidentiality + integrity + authentication
```

```text
Certificate
→ helps authenticate server identity/public-key relationship
```

```text
Forward proxy
→ represents clients
```

```text
Reverse proxy
→ represents servers
```

```text
CONNECT
→ asks an HTTP proxy to create a TCP tunnel
```

```text
HTTPS through CONNECT
→ proxy normally forwards encrypted TLS bytes
```

```text
SOCKS5
→ generic host:port proxying
```

```text
Proxy ≠ VPN

Proxy → application/connection level
VPN   → routing/IP tunnel level
```
