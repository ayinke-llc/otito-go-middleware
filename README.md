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

	return otitoMiddleware.New().Handler(router))
}
```


