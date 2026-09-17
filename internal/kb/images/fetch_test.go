package images

import (
	"context"
	"encoding/base64"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "::1", // loopback
		"10.0.0.1", "172.16.0.5", "192.168.1.1", // private
		"169.254.1.1", "fe80::1", // link-local
		"224.0.0.1", "ff02::1", // multicast
		"0.0.0.0", "::", // unspecified
	}
	for _, s := range blocked {
		if !isBlockedIP(net.ParseIP(s)) {
			t.Errorf("isBlockedIP(%s) = false, want true", s)
		}
	}

	allowed := []string{"8.8.8.8", "1.1.1.1", "93.184.216.34"}
	for _, s := range allowed {
		if isBlockedIP(net.ParseIP(s)) {
			t.Errorf("isBlockedIP(%s) = true, want false", s)
		}
	}
}

func TestDecodeDataURL(t *testing.T) {
	// "hi" base64-encoded.
	got, err := decodeDataURL("data:image/png;base64,aGk=")
	if err != nil {
		t.Fatalf("decodeDataURL: %v", err)
	}
	if string(got) != "hi" {
		t.Errorf("decodeDataURL = %q, want %q", got, "hi")
	}
}

func TestDecodeDataURLRejectsMalformed(t *testing.T) {
	tests := []string{
		"not-a-data-url",
		"data:image/png;base64",          // no comma
		"data:image/png,plaintext",       // not base64
		"data:text/plain;base64,not b64", // invalid base64 payload
	}
	for _, in := range tests {
		if _, err := decodeDataURL(in); err == nil {
			t.Errorf("decodeDataURL(%q) succeeded, want an error", in)
		}
	}
}

func TestStoreDataURL(t *testing.T) {
	store := openTestStore(t)
	personID := testPerson(t, store)

	dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes)
	img, err := store.StoreDataURL(context.Background(), dataURL, personID)
	if err != nil {
		t.Fatalf("StoreDataURL: %v", err)
	}
	if img.Ext != "png" {
		t.Errorf("Ext = %q, want png", img.Ext)
	}
}

func pngHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Write(pngBytes)
}

func TestFetchAndStoreRefusesLoopbackByDefault(t *testing.T) {
	store := openTestStore(t)
	personID := testPerson(t, store)

	ts := httptest.NewServer(http.HandlerFunc(pngHandler))
	defer ts.Close()

	_, err := store.FetchAndStore(context.Background(), ts.URL, personID)
	if err == nil {
		t.Fatal("FetchAndStore against a loopback server succeeded, want a refusal")
	}
}

func TestFetchAndStoreSucceedsWithTestOverride(t *testing.T) {
	store := openTestStore(t)
	personID := testPerson(t, store)

	ts := httptest.NewServer(http.HandlerFunc(pngHandler))
	defer ts.Close()

	setTestFetchOverride(t, ts.Listener.Addr().String())

	img, err := store.FetchAndStore(context.Background(), ts.URL, personID)
	if err != nil {
		t.Fatalf("FetchAndStore: %v", err)
	}
	if img.Ext != "png" {
		t.Errorf("Ext = %q, want png", img.Ext)
	}
}

func TestFetchAndStoreRefusesRedirectToUnlistedLoopbackAddress(t *testing.T) {
	store := openTestStore(t)
	personID := testPerson(t, store)

	target := httptest.NewServer(http.HandlerFunc(pngHandler))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirector.Close()

	// Only the redirector's own address is allow-listed; the target it
	// redirects to is a different loopback address and must still be
	// refused, the same way a real private-network target would be.
	setTestFetchOverride(t, redirector.Listener.Addr().String())

	_, err := store.FetchAndStore(context.Background(), redirector.URL, personID)
	if err == nil {
		t.Fatal("FetchAndStore following a redirect to an unlisted address succeeded, want a refusal")
	}
}

func TestFetchAndStoreRejectsTooManyRedirects(t *testing.T) {
	store := openTestStore(t)
	personID := testPerson(t, store)

	var ts *httptest.Server
	hops := 0
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hops++
		http.Redirect(w, r, ts.URL+"/next", http.StatusFound)
	}))
	defer ts.Close()

	setTestFetchOverride(t, ts.Listener.Addr().String())

	_, err := store.FetchAndStore(context.Background(), ts.URL, personID)
	if err == nil {
		t.Fatal("FetchAndStore with endless redirects succeeded, want a refusal")
	}
	if hops > maxRedirects+2 {
		t.Errorf("followed %d redirects, want at most around %d", hops, maxRedirects)
	}
}

func TestFetchAndStoreTimesOut(t *testing.T) {
	store := openTestStore(t)
	personID := testPerson(t, store)

	old := fetchTimeout
	fetchTimeout = 50 * time.Millisecond
	defer func() { fetchTimeout = old }()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		pngHandler(w, r)
	}))
	defer ts.Close()

	setTestFetchOverride(t, ts.Listener.Addr().String())

	_, err := store.FetchAndStore(context.Background(), ts.URL, personID)
	if err == nil {
		t.Fatal("FetchAndStore against a slow server succeeded, want a timeout error")
	}
}

func TestFetchAndStoreRejectsOversize(t *testing.T) {
	store := openTestStore(t)
	personID := testPerson(t, store)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(make([]byte, MaxBytes+1))
	}))
	defer ts.Close()

	setTestFetchOverride(t, ts.Listener.Addr().String())

	_, err := store.FetchAndStore(context.Background(), ts.URL, personID)
	if err != ErrTooLarge {
		t.Errorf("FetchAndStore(oversize) error = %v, want ErrTooLarge", err)
	}
}

func TestFetchAndStoreRejectsNonImageContent(t *testing.T) {
	store := openTestStore(t)
	personID := testPerson(t, store)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>not a picture</html>"))
	}))
	defer ts.Close()

	setTestFetchOverride(t, ts.Listener.Addr().String())

	_, err := store.FetchAndStore(context.Background(), ts.URL, personID)
	if err != ErrUnsupportedType {
		t.Errorf("FetchAndStore(non-image) error = %v, want ErrUnsupportedType", err)
	}
}

// setTestFetchOverride sets the two env vars that let FetchAndStore reach
// exactly one loopback address in a test, restoring both afterward.
func setTestFetchOverride(t *testing.T, addr string) {
	t.Helper()
	t.Setenv(testModeEnv, "1")
	t.Setenv(testHostEnv, addr)
}
