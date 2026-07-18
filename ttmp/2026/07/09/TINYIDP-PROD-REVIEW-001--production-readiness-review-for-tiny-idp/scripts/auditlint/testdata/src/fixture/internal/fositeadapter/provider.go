package fositeadapter

type Mux struct{}
type Provider struct{}

func (Mux) HandleFunc(string, any) {}

func registerAt(prefix string, mux Mux, p Provider) {
	mux.HandleFunc(prefix+"/device_authorization", p.deviceAuthorization)
	mux.HandleFunc(prefix+"/device", p.deviceVerification)
}

func (Provider) deviceAuthorization() {}
func (Provider) deviceVerification()  {}
