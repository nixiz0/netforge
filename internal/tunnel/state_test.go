package tunnel

import (
	"net"
	"testing"
	"time"
)

func TestEndpointLifecycle(t *testing.T) {
	home := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 61021}
	roamed := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 61099}
	var p endpoint

	if !p.seen(home) {
		t.Fatal("first packet: want a new peer")
	}
	if p.seen(home) {
		t.Fatal("same address again: want the same peer")
	}
	if !p.seen(roamed) {
		t.Fatal("new address: want a new peer (roaming)")
	}
	if _, gone := p.expire(time.Hour); gone {
		t.Fatal("heard just now: must not expire within an hour")
	}
	if addr, gone := p.expire(0); !gone || addr.String() != roamed.String() {
		t.Fatalf("expire(0) = %v, %v; want %v, true", addr, gone, roamed)
	}
	if p.get() != nil {
		t.Fatal("after expiry: want no peer")
	}
	if _, gone := p.expire(0); gone {
		t.Fatal("nothing left to expire")
	}
}
