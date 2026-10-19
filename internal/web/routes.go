package web

import (
	"net/http"

	"nisaba/api"
	"nisaba/internal/web/handlers"
	"nisaba/internal/web/respond"

	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
)

func installRoutes(deps Deps, router *chi.Mux) {
	spec, err := api.GetSpec()
	if err != nil {
		// The spec is embedded at build time, so this is a broken build, not a runtime condition.
		panic("load embedded openapi spec: " + err.Error())
	}

	strict := api.NewStrictHandlerWithOptions(handlers.API{}, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  requestErrorHandler,
		ResponseErrorHandlerFunc: responseErrorHandler,
	})

	router.Get("/healthz", handlers.Health)

	// Unknown /api/* paths answer problem+json, so the frontend catch-all at the root never answers an API path.
	router.Route("/api", func(r chi.Router) {
		r.NotFound(handlers.NotFound)
		r.Route("/v1", func(r chi.Router) {
			if deps.Config.Docs {
				r.Get("/openapi.json", serveSpecJSON(specJSON()))
			}
			// Validation applies only to spec routes; the spec documents above are not operations.
			r.Group(func(r chi.Router) {
				r.Use(oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
					SilenceServersWarning: true,
					ErrorHandlerWithOpts:  validationErrorHandler,
					Options:               openapi3filter.Options{MultiError: true},
				}))
				api.HandlerWithOptions(strict, api.ChiServerOptions{
					BaseRouter:       r,
					ErrorHandlerFunc: routerErrorHandler,
				})
			})
		})
		if deps.Config.Docs {
			r.Route("/docs", func(r chi.Router) {
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					respond.To(w, r).Redirect("/api/docs/scalar/")
				})
				files := docsFiles()
				r.With(immutable).Handle("/{ui}/{version:[0-9]+\\.[0-9]+\\.[0-9]+}/*", files)
				r.Handle("/*", files)
			})
		}
	})
	if deps.Config.FrontendDevServer != nil {
		router.Get("/*", proxyFrontend(deps.Config.FrontendDevServer))
	} else {
		router.Get("/*", serveFrontend(frontendFS()))
	}
}
