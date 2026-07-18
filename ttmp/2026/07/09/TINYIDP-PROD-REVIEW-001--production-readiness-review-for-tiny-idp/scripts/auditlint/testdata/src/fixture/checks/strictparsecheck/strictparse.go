package strictparsecheck

import "strconv"

func unsafeMaxAge(raw string) bool {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return true // want "numeric parse failure returns true in unsafeMaxAge"
	}
	return value > 0
}

func safeMaxAge(raw string) (int64, error) {
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}
