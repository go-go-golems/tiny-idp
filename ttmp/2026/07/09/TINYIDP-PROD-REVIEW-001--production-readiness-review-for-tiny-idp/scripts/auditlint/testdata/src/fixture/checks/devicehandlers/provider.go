package devicehandlers

type Mux struct{}
type Provider struct{}

func (Mux) HandleFunc(string, any) {}

func registerAt(mux Mux, p Provider) {
	mux.HandleFunc("/device_authorization", p.deviceAuthorization)
	mux.HandleFunc("/device", p.deviceVerification)
}

func (Provider) deviceAuthorization() {}
func (Provider) deviceVerification()  {}
