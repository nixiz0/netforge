package tunnel

import (
	"crypto/ecdh"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sync/atomic"
)

// Peer is one VPN peer this end accepts: who it is (its public key) and its address inside the VPN.
//
// Theory: _networking-lessons/9-multi-peer-networking.md#peer-identity-vpn-address-endpoint
type Peer struct {
	Key  *ecdh.PublicKey
	Addr netip.Addr // VPN address: inner packets for it are sent to this peer (server mode)
}

// peerState is one row of the peer table: identity and VPN address, handshake and session keys,
// replay state (inside the sealers), and the current endpoint with its last activity.
//
// Theory: _networking-lessons/9-multi-peer-networking.md#peer-table
// Theory: _networking-lessons/9-multi-peer-networking.md#per-peer-security-state
type peerState struct {
	Peer
	hs      *handshake
	session atomic.Pointer[sealer]
	ep      endpoint
}

// peerTable is built once at start and never changes, so reading it needs no lock: each row
// protects its own mutable state.
type peerTable struct {
	all    []*peerState
	byAddr map[netip.Addr]*peerState
}

func newPeerTable(priv *ecdh.PrivateKey, peers []Peer, isClient bool) (*peerTable, error) {
	t := &peerTable{byAddr: map[netip.Addr]*peerState{}}
	for _, p := range peers {
		hs, err := newHandshake(priv, p.Key, isClient)
		if err != nil {
			return nil, err
		}
		row := &peerState{Peer: p, hs: hs}
		t.all = append(t.all, row)
		if p.Addr.IsValid() {
			if _, dup := t.byAddr[p.Addr]; dup {
				return nil, fmt.Errorf("two peers with VPN address %s", p.Addr)
			}
			t.byAddr[p.Addr] = row
		}
	}
	return t, nil
}

// route picks the peer for an inner IP packet by its destination address: peer routing.
// Each peer owns one exact address here, so a map lookup is the whole routing table.
//
// Theory: _networking-lessons/9-multi-peer-networking.md#peer-routing
func (t *peerTable) route(ip []byte) *peerState {
	dst, ok := innerDst(ip)
	if !ok {
		return nil
	}
	return t.byAddr[dst]
}

// identify finds the peer whose keys open b, and returns it with the opened packet. It tries the
// peer last seen at from first, then all the others (so a roaming peer is still recognized).
// Trying keys one by one is fine for a few peers; WireGuard puts a receiver index in each packet
// to find the peer in a single lookup.
func (t *peerTable) identify(b []byte, from net.Addr) (*peerState, *sealer, packet, error) {
	handshakeMsg := len(b) > 1 && (b[1] == typeInit || b[1] == typeResp)
	order := make([]*peerState, 0, len(t.all))
	for _, p := range t.all {
		if a := p.ep.get(); a != nil && a.String() == from.String() {
			order = append([]*peerState{p}, order...)
		} else {
			order = append(order, p)
		}
	}
	err := errors.New("no session yet")
	for _, p := range order {
		keys := p.session.Load()
		if handshakeMsg {
			keys = p.hs.static // handshake messages are sealed with the static keys
		}
		if keys == nil {
			continue
		}
		pkt, e := keys.open(b)
		if e == nil {
			return p, keys, pkt, nil
		}
		// Any error other than a tag mismatch came from the right keys (a replay, say): stop.
		if !errors.Is(e, errAuth) {
			return nil, nil, packet{}, e
		}
		err = e
	}
	return nil, nil, packet{}, err
}

// innerDst reads the destination address of an IPv4 packet.
func innerDst(ip []byte) (netip.Addr, bool) {
	if len(ip) < 20 || ip[0]>>4 != 4 {
		return netip.Addr{}, false
	}
	return netip.AddrFrom4([4]byte(ip[16:20])), true
}
