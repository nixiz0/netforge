package tunnel

import (
	"net"
	"sync"
	"time"
)

// Keepalives: the client sends a tiny packet every keepaliveInterval, even when idle, so that NAT
// and firewall state along the path does not expire. A peer silent for peerTimeout is considered gone.
//
// Theory: _networking-lessons/5-tunneling.md#keepalives
// Theory: _networking-lessons/4-advanced-network-concepts.md#nat-mappings-from-outbound-traffic
const (
	keepaliveInterval = 5 * time.Second
	peerTimeout       = 3 * keepaliveInterval
)

// endpoint is the tunnel state about the other end's current real address (IP:port, its endpoint) and
// when we last heard from it. Several goroutines use it, so every method takes the mutex.
//
// Theory: _networking-lessons/9-multi-peer-networking.md#peer-identity-vpn-address-endpoint
//
// Theory: _networking-lessons/5-tunneling.md#tunnel-state
type endpoint struct {
	mu       sync.Mutex
	addr     net.Addr
	lastSeen time.Time
}

// seen records a packet from addr. It returns true when the address is new: the first one, one that
// had expired, or the same peer arriving from a new address (roaming).
//
// Theory: _networking-lessons/7-vpn-architecture.md#endpoint-changes--roaming
func (e *endpoint) seen(addr net.Addr) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	isNew := e.addr == nil || e.addr.String() != addr.String()
	e.addr, e.lastSeen = addr, time.Now()
	return isNew
}

// get returns the current address, or nil when there is none.
func (e *endpoint) get() net.Addr {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.addr
}

// expire forgets the address if it has been silent for longer than timeout.
// It returns the forgotten address and true when it just did.
func (e *endpoint) expire(timeout time.Duration) (net.Addr, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.addr == nil || time.Since(e.lastSeen) < timeout {
		return nil, false
	}
	gone := e.addr
	e.addr = nil
	return gone, true
}
