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

// peer is the tunnel state about the other end: its address and when we last heard from it.
// Several goroutines use it, so every method takes the mutex.
//
// Theory: _networking-lessons/5-tunneling.md#tunnel-state
type peer struct {
	mu       sync.Mutex
	addr     net.Addr
	lastSeen time.Time
}

// seen records a packet from addr. It returns true when this is a new peer: the first one, one that
// had expired, or the same peer arriving from a new address (roaming).
//
// Theory: _networking-lessons/7-vpn-architecture.md#endpoint-changes--roaming
func (p *peer) seen(addr net.Addr) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	isNew := p.addr == nil || p.addr.String() != addr.String()
	p.addr, p.lastSeen = addr, time.Now()
	return isNew
}

// get returns the peer's current address, or nil when there is none.
func (p *peer) get() net.Addr {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.addr
}

// expire forgets the peer if it has been silent for longer than timeout.
// It returns the forgotten address and true when it just did.
func (p *peer) expire(timeout time.Duration) (net.Addr, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.addr == nil || time.Since(p.lastSeen) < timeout {
		return nil, false
	}
	gone := p.addr
	p.addr = nil
	return gone, true
}
