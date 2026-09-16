package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOriginCheck(t *testing.T) {
	ok := OriginCheck(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	cases := []struct {
		name       string
		method     string
		host       string
		origin     string
		referer    string
		wantStatus int
	}{
		{"GET is never checked", http.MethodGet, "comphq.example", "https://evil.example", "", http.StatusOK},
		{"matching origin", http.MethodPost, "comphq.example", "https://comphq.example", "", http.StatusOK},
		{"matching origin with port", http.MethodPost, "comphq.example:8080", "http://comphq.example:8080", "", http.StatusOK},
		{"mismatched origin", http.MethodPost, "comphq.example", "https://evil.example", "", http.StatusForbidden},
		{"falls back to referer", http.MethodPost, "comphq.example", "", "https://comphq.example/page", http.StatusOK},
		{"mismatched referer", http.MethodPost, "comphq.example", "", "https://evil.example/page", http.StatusForbidden},
		{"neither header present", http.MethodPost, "comphq.example", "", "", http.StatusForbidden},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(c.method, "/tasks/1/move", nil)
			req.Host = c.host
			if c.origin != "" {
				req.Header.Set("Origin", c.origin)
			}
			if c.referer != "" {
				req.Header.Set("Referer", c.referer)
			}
			rec := httptest.NewRecorder()
			ok.ServeHTTP(rec, req)
			if rec.Code != c.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, c.wantStatus)
			}
		})
	}
}
