package devicetransition

type Store struct{}

func (Store) Update() {}

func DeviceTransition(s Store) { s.Update() } // want `device-grant transition must use a named DeviceGrant operation, not generic Update`
