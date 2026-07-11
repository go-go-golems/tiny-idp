package clockcheck

import "time"

func beginAuthorize() time.Time {
	return time.Now() // want "security state transition beginAuthorize reads time.Now directly"
}

func unrelated() time.Time {
	return time.Now()
}
