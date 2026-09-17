package html

import (
	"reflect"
	"testing"
)

func TestRewriteExternalImagesSuccess(t *testing.T) {
	in := `<p>Before</p><img src="https://example.com/a.png" alt="A"><p>After</p>`
	fetch := func(src string) (string, bool) {
		if src == "https://example.com/a.png" {
			return "/images/deadbeef.png", true
		}
		t.Fatalf("unexpected fetch(%q)", src)
		return "", false
	}

	got, failures := RewriteExternalImages(in, fetch)
	want := `<p>Before</p><img src="/images/deadbeef.png" alt="A"/><p>After</p>`
	if got != want {
		t.Errorf("RewriteExternalImages =\n%q\nwant\n%q", got, want)
	}
	if failures != nil {
		t.Errorf("failures = %v, want nil", failures)
	}
}

func TestRewriteExternalImagesFailure(t *testing.T) {
	in := `<p>Before</p><img src="https://example.com/a.png" alt="A"><p>After</p>`
	fetch := func(src string) (string, bool) { return "", false }

	got, failures := RewriteExternalImages(in, fetch)
	want := `<p>Before</p><div data-missing-kind="external" data-missing-src="https://example.com/a.png"></div><p>After</p>`
	if got != want {
		t.Errorf("RewriteExternalImages =\n%q\nwant\n%q", got, want)
	}
	if !reflect.DeepEqual(failures, []string{"https://example.com/a.png"}) {
		t.Errorf("failures = %v, want [https://example.com/a.png]", failures)
	}
}

func TestRewriteExternalImagesSkipsAlreadyLocal(t *testing.T) {
	in := `<img src="/images/already.png" alt="A">`
	called := false
	fetch := func(src string) (string, bool) {
		called = true
		return "", false
	}

	got, failures := RewriteExternalImages(in, fetch)
	if called {
		t.Error("fetch was called for an already-local image")
	}
	if want := `<img src="/images/already.png" alt="A"/>`; got != want {
		t.Errorf("RewriteExternalImages = %q, want %q", got, want)
	}
	if failures != nil {
		t.Errorf("failures = %v, want nil", failures)
	}
}

func TestRewriteExternalImagesMultipleInDocumentOrder(t *testing.T) {
	in := `<img src="https://a.example/1.png"><img src="/images/local.png"><img src="https://a.example/2.png">`
	fetch := func(src string) (string, bool) {
		if src == "https://a.example/1.png" {
			return "/images/one.png", true
		}
		return "", false
	}

	got, failures := RewriteExternalImages(in, fetch)
	want := `<img src="/images/one.png"/><img src="/images/local.png"/><div data-missing-kind="external" data-missing-src="https://a.example/2.png"></div>`
	if got != want {
		t.Errorf("RewriteExternalImages =\n%q\nwant\n%q", got, want)
	}
	if !reflect.DeepEqual(failures, []string{"https://a.example/2.png"}) {
		t.Errorf("failures = %v, want [https://a.example/2.png]", failures)
	}
}
