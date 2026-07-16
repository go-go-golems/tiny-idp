package devicecheck

type Event struct{ Fields map[string]string }
type Request struct{}
type Mux struct{}
type Provider struct{}
type Store struct{}

func (Request) ParseForm() error   { return nil }
func MaxBytesReader()              {}
func (Mux) HandleFunc(string, any) {}
func (Store) Update()              {}
func (Store) DecideDeviceGrant()   {}

func unsafeAudit() {
	_ = Event{Fields: map[string]string{"device_code": "raw"}} // want `device audit field "device_code" may contain secret credential material`
}

func safeAudit() { _ = Event{Fields: map[string]string{"scope_count": "2"}} }

func deviceAuthorization(r Request)        { _ = r.ParseForm() } // want `device handler deviceAuthorization parses attacker input without http.MaxBytesReader`
func completeDeviceVerification(r Request) { MaxBytesReader(); _ = r.ParseForm() }

func DeviceTransition(s Store) { s.Update() } // want `device-grant transition must use a named DeviceGrant operation, not generic Update`
func DeviceDecision(s Store)   { s.DecideDeviceGrant() }

func registerAt(mux Mux, p Provider) {
	mux.HandleFunc("/device_authorization", p.deviceAuthorization)
	mux.HandleFunc("/device", p.deviceVerification)
}

func (Provider) deviceAuthorization() {}
func (Provider) deviceVerification()  {}
