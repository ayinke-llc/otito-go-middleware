package otito

import (
	"net/http"
	"strings"
)

var xForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
var xRealIP = http.CanonicalHeaderKey("X-Real-IP")

type IPStrategy uint

const (
	// CloudflareStrategy is used for apps running behind cloudflare
	CloudflareStrategy IPStrategy = iota + 1
	// Uses the standard X-Forwarded-For or X-Real-IP http header to find
	// the ip
	ForwardedOrRealIPStrategy
	// please don't use this in prod. Maybe when running locally only
	RemoteHeaderStrategy
)

func getIP(r *http.Request, strategy IPStrategy) string {

	var ip string

	switch strategy {
	case CloudflareStrategy:
		return r.Header.Get(http.CanonicalHeaderKey("CF-Connecting-IP"))
	case ForwardedOrRealIPStrategy:
		if xff := r.Header.Get(xForwardedFor); xff != "" {
			i := strings.Index(xff, ", ")

			if i == -1 {
				i = len(xff)
			}

			return xff[:i]
		}

		return r.Header.Get(xRealIP)
	case RemoteHeaderStrategy:
		return r.RemoteAddr

	default:
		return ""
	}
}
