package httprouter

import (
	"net/http"

	"github.com/go-chi/chi/v4"
)

// Router managed HTTP communication with the GDPR mapping logic
type Router struct {
	chi.Router
}

// New returns a new Router
func New() *Router {
	router := &Router{
		Router: chi.NewRouter(),
	}

	router.Get("/hello", router.hello)

	return router
}

func (r *Router) hello(respw http.ResponseWriter, reqr *http.Request) {
	_, _ = respw.Write([]byte(`{"hello": "world"}`))
}
