package scratch

import (
	"bytes"
	"testing"
)

func TestGrow(t *testing.T) {
	sb := &Buf{}
	sb.Write([]byte{1, 2, 3})
	need := 321
	sb.Grow(need)
	avail := sb.Cap() - sb.Len()
	if avail < need {
		t.Fatalf("Grow(%d) results in cap-len=%d but it should be at least %d", need, avail, need)
	}
}

func TestTail(t *testing.T) {
	sb := &Buf{}
	sb.Write([]byte{1, 2, 3})
	s := sb.Tail(1)
	if len(s) != 1 {
		t.Fatalf("Tail(1) results in a slice of len %d", len(s))
	}
	if s[0] != 0 {
		t.Fatalf("Tail(1) returns %#v instead of %#v", s, []byte{0})
	}
	if p, q := sb.Bytes(), []byte{1, 2, 3, 0}; !bytes.Equal(p, q) {
		t.Fatalf("Tail(1) results in Bytes() %#v instead of %#v", p, q)
	}
}
