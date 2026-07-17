package devicebounded

import "net/http"

func deviceAuthorization(r *http.Request) { _ = r.ParseForm() } // want `device handler deviceAuthorization parses attacker input without http.MaxBytesReader`

func completeDeviceVerification(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	_ = r.ParseForm()
}
