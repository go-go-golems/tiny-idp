package fositeadapter

import (
	"fmt"
	"strings"
	"time"

	idpstore "github.com/go-go-golems/tiny-idp/pkg/idpstore"
)

const defaultBrowserContextCookieName = "tinyidp_browser_context"

// AccountChooserConfig controls provider-owned multi-account browser state.
// It is disabled by default because remembered account labels can reveal prior
// use of the browser. A host must deliberately opt in and provide a label
// policy before password logins are remembered.
type AccountChooserConfig struct {
	Enabled                 bool
	ContextCookieName       string
	ContextTTL              time.Duration
	MaxRememberedAccounts   int
	RememberOnPasswordLogin bool
	DisplayLabel            func(idpstore.User) (string, error)
}

func (c *AccountChooserConfig) normalize() error {
	if !c.Enabled {
		return nil
	}
	if c.ContextCookieName == "" {
		c.ContextCookieName = defaultBrowserContextCookieName
	}
	if !validCookieName(c.ContextCookieName) {
		return fmt.Errorf("fositeadapter: browser context cookie name is invalid")
	}
	if c.ContextTTL == 0 {
		c.ContextTTL = 30 * 24 * time.Hour
	}
	if c.ContextTTL <= 0 {
		return fmt.Errorf("fositeadapter: browser context TTL must be positive")
	}
	if c.MaxRememberedAccounts == 0 {
		c.MaxRememberedAccounts = 5
	}
	if c.MaxRememberedAccounts < 1 || c.MaxRememberedAccounts > 20 {
		return fmt.Errorf("fositeadapter: maximum remembered accounts must be between 1 and 20")
	}
	if c.RememberOnPasswordLogin && c.DisplayLabel == nil {
		return fmt.Errorf("fositeadapter: remembered password logins require an account display-label policy")
	}
	return nil
}

func (c AccountChooserConfig) labelFor(user idpstore.User) (string, error) {
	if c.DisplayLabel == nil {
		return "", fmt.Errorf("account display-label policy is unavailable")
	}
	label, err := c.DisplayLabel(user)
	if err != nil {
		return "", err
	}
	label = strings.TrimSpace(label)
	if label == "" {
		return "", fmt.Errorf("account display-label policy returned an empty label")
	}
	if len([]rune(label)) > 120 {
		return "", fmt.Errorf("account display-label policy returned a label longer than 120 characters")
	}
	return label, nil
}
