package randcheck

import "crypto/rand"

func randomID() {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // want `error from crypto/rand.Read is ignored`
}
