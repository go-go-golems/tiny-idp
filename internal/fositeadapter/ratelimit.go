package fositeadapter

import (
	"context"

	"github.com/manuel/tinyidp/pkg/idp"
)

type AllowAllRateLimiter struct{}

var _ idp.RateLimiter = AllowAllRateLimiter{}

func (AllowAllRateLimiter) Allow(context.Context, string) bool { return true }
func (AllowAllRateLimiter) ProductionReady() bool              { return false }
