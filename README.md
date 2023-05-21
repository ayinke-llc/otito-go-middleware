### Golang Middleware for Otito


#### Usage

```go

import (
	otitoMiddleware "github.com/ayinke-llc/otito-go-middleware"
	chi "github.com/go-chi/chi/v5"
)

func main() {
	router := chi.NewRouter()

	router.Use(middleware.AllowContentType("application/json"))
	router.Use(middleware.RequestID)
	router.Use(otelchi.Middleware("http-router", otelchi.WithChiRoutes(router)))

	// do not ignore in real life
	handler, _ := otitoMiddleware.New(otitoMiddleware.WithAPIKey("sk_KEY"),
		otitoMiddleware.WithAppIDFn(func(r *http.Request) string {
		return `return an ID patterning to the current user. You can get
		from the current request session really depending on how your
		app is structed` }),
		otitoMiddleware.WithIPStrategy(otitoMiddleware.RemoteHeaderStrategy),
		otitoMiddleware.WithNumberOfMessagesBeforePublishing(100)) // publish to the ingester every 100 http requests

	return otitoMiddleware.New().Handler(router))
}
```


