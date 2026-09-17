package html

import "testing"

func TestHasUnresolvedContent(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"clean body", `<p>Nothing wrong here.</p><img src="/images/abc123.png" alt="fine">`, false},
		{"no images at all", `<p>Just text.</p>`, false},
		{"missing-src placeholder", `<p>See:</p><div data-missing-kind="external" data-missing-src="https://example.com/a.png"></div>`, true},
		{"missing-kind placeholder without a src", `<div data-missing-kind="chart"></div>`, true},
		{"image not under /images/", `<img src="https://example.com/a.png" alt="foreign">`, true},
		{"image with no src at all", `<img alt="broken">`, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := HasUnresolvedContent(c.body); got != c.want {
				t.Errorf("HasUnresolvedContent(%q) = %v, want %v", c.body, got, c.want)
			}
		})
	}
}
