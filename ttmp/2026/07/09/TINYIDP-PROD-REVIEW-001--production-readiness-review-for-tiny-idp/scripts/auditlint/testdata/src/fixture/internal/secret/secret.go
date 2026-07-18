package secret

type Store interface {
	Save() error
}
