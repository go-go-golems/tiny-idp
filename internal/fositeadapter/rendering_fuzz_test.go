package fositeadapter

import "testing"

func FuzzBoundedInteractionBuffer(f *testing.F) {
	f.Add(256, []byte("hello"), []byte(" world"))
	f.Add(4, []byte("abcd"), []byte("e"))
	f.Add(0, []byte("x"), []byte{})
	f.Fuzz(func(t *testing.T, limit int, first, second []byte) {
		if limit < 0 {
			limit = -limit
		}
		if limit > 1<<20 {
			limit %= 1 << 20
		}
		buffer := &boundedInteractionBuffer{limit: limit}
		_, firstErr := buffer.Write(first)
		_, secondErr := buffer.Write(second)
		if buffer.Len() > limit {
			t.Fatalf("buffer length %d exceeded limit %d", buffer.Len(), limit)
		}
		overflowExpected := len(first)+len(second) > limit
		if overflowExpected != buffer.overflowed {
			t.Fatalf("overflow=%v want=%v firstErr=%v secondErr=%v", buffer.overflowed, overflowExpected, firstErr, secondErr)
		}
	})
}
