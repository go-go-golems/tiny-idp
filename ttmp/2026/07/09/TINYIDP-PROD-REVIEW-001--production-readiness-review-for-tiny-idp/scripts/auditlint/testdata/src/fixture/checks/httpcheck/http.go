package httpcheck

import "net/http"

func serve() error {
	return http.ListenAndServe(":8080", http.NewServeMux()) // want `http.ListenAndServe uses a zero-value Server`
}
