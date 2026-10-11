package tunnel

// Replay protection. A recorded packet sent again later still has a valid tag: AEAD proves who
// sealed it, not when. So the receiver remembers which nonce counters it already accepted and
// refuses any second copy.
//
// It keeps the highest counter seen plus a 64-bit bitmap of the 64 counters just below it, so
// packets that UDP delivers slightly out of order are still accepted, exactly once:
//
//	top = 100   seen = ...1011   → 100, 99 and 97 accepted; 98 still welcome; 36 and below too old
//
// Theory: _networking-lessons/8-security-cryptography.md#replay-attacks
// Theory: _networking-lessons/8-security-cryptography.md#replay-protection
const windowSize = 64

type replayWindow struct {
	prefix [4]byte // the sender's nonce prefix: a new one means the sender restarted
	top    uint64  // highest counter accepted, 0 = nothing yet (counters start at 1)
	seen   uint64  // bit i set: counter top-i was accepted
}

// accept reports whether this nonce is new, and records it. Call it only for packets whose tag
// is valid, or forged counters could push the window forward and block the real peer.
func (w *replayWindow) accept(prefix [4]byte, counter uint64) bool {
	if prefix != w.prefix || w.top == 0 {
		// The peer restarted (fresh prefix, counter back to 1): start a new window.
		w.prefix, w.top, w.seen = prefix, counter, 1
		return true
	}
	switch {
	case counter > w.top: // newer than anything so far: slide the window up
		if shift := counter - w.top; shift < windowSize {
			w.seen = w.seen<<shift | 1
		} else {
			w.seen = 1
		}
		w.top = counter
		return true
	case w.top-counter >= windowSize: // fell out of the window: can't tell, so refuse
		return false
	default: // inside the window: accept once
		bit := uint64(1) << (w.top - counter)
		if w.seen&bit != 0 {
			return false
		}
		w.seen |= bit
		return true
	}
}
