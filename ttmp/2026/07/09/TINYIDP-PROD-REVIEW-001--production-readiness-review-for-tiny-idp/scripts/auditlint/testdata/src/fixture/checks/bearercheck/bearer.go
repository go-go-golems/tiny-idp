package bearercheck

import (
	"net/http"

	"github.com/ory/fosite"
)

func userinfo(request *http.Request) string {
	return fosite.AccessTokenFromRequest(request) // want "fosite.AccessTokenFromRequest accepts query and form bearer tokens"
}
