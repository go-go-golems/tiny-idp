package config

type CookieConfig struct {
	Secure   bool
	SameSite string // want `exported configuration field CookieConfig.SameSite is never read`
}

func secure(c CookieConfig) bool { return c.Secure }
