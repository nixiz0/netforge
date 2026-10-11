package tunnel

import "testing"

func TestReplayWindow(t *testing.T) {
	a, b := [4]byte{1}, [4]byte{2}
	var w replayWindow
	steps := []struct {
		name    string
		prefix  [4]byte
		counter uint64
		want    bool
	}{
		{"first packet", a, 1, true},
		{"next", a, 2, true},
		{"same again", a, 2, false},
		{"jump ahead", a, 10, true},
		{"late but new", a, 5, true},
		{"late, already seen", a, 5, false},
		{"far ahead", a, 100, true},
		{"edge of the window", a, 100 - windowSize + 1, true},
		{"just out of the window", a, 100 - windowSize, false},
		{"peer restarted", b, 1, true},
		{"duplicate after the restart", b, 1, false},
		// The window alone only remembers one run: a packet recorded before the restart has another
		// prefix, so it looks like yet another restart. In the VPN it never gets here: it was sealed
		// with the previous session keys, so it fails the tag first (TestHandshake).
		{"packet from the old run", a, 2, true},
	}
	for _, s := range steps { // in order: each step depends on the ones before
		if got := w.accept(s.prefix, s.counter); got != s.want {
			t.Fatalf("%s: accept(%v, %d) = %v, want %v", s.name, s.prefix, s.counter, got, s.want)
		}
	}
}

// The attack itself: the exact same valid packet, delivered twice.
func TestOpenRejectsReplay(t *testing.T) {
	a, b := testSealer(t, 1), testSealer(t, 1)
	wire := a.seal(typeKeepalive, nil)
	if _, err := b.open(wire); err != nil {
		t.Fatal(err)
	}
	if _, err := b.open(wire); err == nil {
		t.Fatal("the replayed packet was accepted")
	}
}
