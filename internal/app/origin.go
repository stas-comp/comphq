package app

import (
	"net/http"
	"net/url"
)

// OriginCheck enforces SPEC B4 "request safety": every state-changing
// request must be a POST whose Origin header (or, failing that, Referer)
// host matches Host, or it gets a 403. This isn't about roles or logins
// (SPEC A3: "anyone can do anything") — it's a cross-site request forgery
// guard, since the person cookie alone is "attribution only, not security."
func OriginCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			next.ServeHTTP(w, r)
			return
		}

		source := r.Header.Get("Origin")
		if source == "" {
			source = r.Header.Get("Referer")
		}
		if source == "" || !sameHost(source, r.Host) {
			http.Error(w, "This request looks like it came from somewhere else, so it was refused.", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sameHost(originOrReferer, host string) bool {
	u, err := url.Parse(originOrReferer)
	if err != nil {
		return false
	}
	return u.Host == host
}
