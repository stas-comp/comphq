package images

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"
	"time"
)

const (
	maxRedirects = 3
	testModeEnv  = "COMPHQ_TEST_MODE"
	testHostEnv  = "COMPHQ_TEST_ALLOW_FETCH_HOST"
)

// fetchTimeout is a var, not a const, so TestFetchAndStoreTimesOut can
// shrink it rather than making every test run wait out a real 10s
// timeout.
var fetchTimeout = 10 * time.Second

// ErrBlockedAddress is returned when a fetch would connect to a loopback,
// private, link-local, multicast or unspecified address (SPEC B4: refuse
// SSRF against the NAS's own network).
var ErrBlockedAddress = errors.New("that address isn't allowed")

// FetchAndStore fetches an http(s) URL server-side and stores the result,
// refusing anything that looks like it targets the NAS's own network
// (SPEC B4, gate 1.19). The 20 MB cap and content sniffing are Save's job,
// applied uniformly to every image regardless of where its bytes came
// from.
func (s *Store) FetchAndStore(ctx context.Context, rawURL string, personID int64) (Image, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return Image{}, fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return Image{}, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}

	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return Image{}, err
	}
	resp, err := fetchClient().Do(req)
	if err != nil {
		return Image{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Image{}, fmt.Errorf("unexpected status %s", resp.Status)
	}

	return s.Save(ctx, resp.Body, personID)
}

// StoreDataURL decodes a data: URL and stores its bytes (SPEC B4: "data:
// URLs are decoded and stored").
func (s *Store) StoreDataURL(ctx context.Context, dataURL string, personID int64) (Image, error) {
	data, err := decodeDataURL(dataURL)
	if err != nil {
		return Image{}, err
	}
	return s.Save(ctx, bytes.NewReader(data), personID)
}

func decodeDataURL(raw string) ([]byte, error) {
	rest, ok := strings.CutPrefix(raw, "data:")
	if !ok {
		return nil, errors.New("not a data: URL")
	}
	meta, payload, ok := strings.Cut(rest, ",")
	if !ok {
		return nil, errors.New("malformed data URL")
	}
	if !strings.HasSuffix(meta, ";base64") {
		return nil, errors.New("only base64-encoded data URLs are supported")
	}
	return base64.StdEncoding.DecodeString(payload)
}

// fetchClient builds an http.Client whose dialer refuses to connect to a
// loopback, private, link-local, multicast or unspecified address, checked
// against the address actually being dialed (post-DNS-resolution), not
// just the URL's hostname — the standard defence against DNS rebinding.
// allowedTestHost lets exactly one address through this check, and only
// when COMPHQ_TEST_MODE=1: the Playwright-started stub image server E2E
// tests publish against necessarily listens on a loopback address, and
// this is the one, narrow, test-only way to reach it.
func fetchClient() *http.Client {
	allowedTestHost := testAllowedFetchHost()
	dialer := &net.Dialer{
		Timeout: fetchTimeout,
		Control: func(network, address string, _ syscall.RawConn) error {
			if allowedTestHost != "" && address == allowedTestHost {
				return nil
			}
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return fmt.Errorf("could not parse address %q", address)
			}
			if isBlockedIP(ip) {
				return ErrBlockedAddress
			}
			return nil
		},
	}
	return &http.Client{
		Timeout:   fetchTimeout,
		Transport: &http.Transport{DialContext: dialer.DialContext},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
}

func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

func testAllowedFetchHost() string {
	if os.Getenv(testModeEnv) != "1" {
		return ""
	}
	return os.Getenv(testHostEnv)
}
