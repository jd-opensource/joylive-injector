package route

import (
	"net/http"
)

func init() {
	RegisterHandler(HandleFunc{
		Path:   "/healthz",
		Method: http.MethodGet,
		Func:   ok,
	})
	RegisterHandler(HandleFunc{
		Path:   "/livez",
		Method: http.MethodGet,
		Func:   ok,
	})
	RegisterHandler(HandleFunc{
		Path:   "/readyz",
		Method: http.MethodGet,
		Func:   ok,
	})
	RegisterHandler(HandleFunc{
		Path:   "/health",
		Method: http.MethodGet,
		Func:   ok,
	})
}

func ok(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
