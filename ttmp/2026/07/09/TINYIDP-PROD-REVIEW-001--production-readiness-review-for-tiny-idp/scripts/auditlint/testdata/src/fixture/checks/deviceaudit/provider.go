package deviceaudit

type Event struct{ Fields map[string]string }

func unsafeAudit() {
	_ = Event{Fields: map[string]string{"device_code": "raw"}} // want `device audit field "device_code" may contain secret credential material`
}
