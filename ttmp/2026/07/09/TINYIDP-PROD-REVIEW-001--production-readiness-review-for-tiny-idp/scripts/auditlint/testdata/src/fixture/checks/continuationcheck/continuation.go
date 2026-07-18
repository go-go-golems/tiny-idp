package continuationcheck

import "net/http"

func resumeAuthorize(request *http.Request) {
	_ = request.PostForm.Get("state") // want "authorization resume reads browser-owned protocol field"
	_ = request.PostForm.Get("login")
}
