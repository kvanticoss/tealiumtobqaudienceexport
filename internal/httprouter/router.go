package httprouter

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/kvanticoss/tealiumtobqaudienceexport/internal/models"

	"cloud.google.com/go/bigquery"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v4"
	"github.com/go-chi/render"
)

// TODO: Split up router and controller logic

// Router managed HTTP communication with the GDPR mapping logic
type Router struct {
	chi.Router

	bq *bigquery.Inserter
}

// New returns a new Router
func New(bq *bigquery.Inserter, allowedKeys map[string]string) *Router {
	rtr := &Router{
		Router: chi.NewRouter(),
		bq:     bq,
	}

	rtr.Use(middleware.BasicAuth("bigquery_export", allowedKeys))

	rtr.Post("/push_audiences_to_bigquery", rtr.stream)

	return rtr
}

func (rtr *Router) stream(w http.ResponseWriter, r *http.Request) {
	v := &models.AudienceTable{}
	if err := render.DefaultDecoder(r, v); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		render.DefaultResponder(w, r, err.Error())
		return
	}
	v.Updated = time.Now()
	if v.Properties.TealiumVisitorId == "" {
		v.Properties.TealiumVisitorId = chi.URLParam(r, "tealium_visitor_id")
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), time.Second*10)
	defer cancelFn()

	if err := rtr.bq.Put(ctx, v); err != nil {
		log.Printf("Error streaming record to bigquery:%v", err)
		render.DefaultResponder(w, r, err.Error())
	}

	render.DefaultResponder(w, r, "ok")
}
