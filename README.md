### Golang Middleware for Otito


#### Usage

```go

import (
	otitoMiddleware "github.com/ayinke-llc/otito-go-middleware"
	chi "github.com/go-chi/chi/v5"
)

func main() {
	router := chi.NewRouter()

	// router.Use(middleware.AllowContentType("application/json"))
	// router.Use(middleware.RequestID)
	// router.Use(otelchi.Middleware("http-router", otelchi.WithChiRoutes(router)))

	// let's assume this middleware stores the user details in the request context
	router.Use(requireAuthentication(userRepo, logger))

	// by default, Authorization headers values are masked and not sent to
	// ingester/API. You can configure more headers to strip by using the WithHeadersToStrip function
	handler, err := otitoMiddleware.New(otitoMiddleware.WithAPIKey(config.Global().Otito.Key),
		otitoMiddleware.WithAppIDFn(func(r *http.Request) string {
			// read the context value here
			// this allows you configure and map each request to the right user
			// if you provide an empty string, this request will not
			// be stored as it won't be mapped to an app
			// If you are not using a context, get the authenticated
			// user from the db here. but a context makes sense as
			// you can skip extra calls

			// Essentially you just need to map this request to the
			// right user regardless of what method you use

			user, ok := r.context().value(userCtx).(*User)
			if !ok {
			    return ""
			}

			return user.OtitoAppID
		}),
		otitoMiddleware.WithFilterFn(func(r *http.Request) bool {
			// let's assume you want to skip storing some request
			return !strings.Contains(r.URL.Path, "auth")
		}),
		otitoMiddleware.WithIPStrategy(otitoMiddleware.RemoteHeaderStrategy),
		otitoMiddleware.WithNumberOfMessagesBeforePublishing(10))

	if err != nil {
		panic(err.Error())
	}

	router.Use(handler.Handler)
}
```


