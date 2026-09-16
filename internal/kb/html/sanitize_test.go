package html

import "testing"

func TestSanitize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "scripts and handlers removed",
			in:   `<p onclick="alert(1)">Hello</p><script>alert(1)</script>`,
			want: `<p>Hello</p>`,
		},
		{
			name: "styles, classes and fonts stripped",
			in:   `<p style="color:red" class="big"><font color="red">Hi</font></p>`,
			want: `<p>Hi</p>`,
		},
		{
			name: "id stripped",
			in:   `<p id="para-1">Hi</p>`,
			want: `<p>Hi</p>`,
		},
		{
			name: "external image dropped",
			in:   `<img src="https://evil.example/x.png" alt="x">`,
			want: ``,
		},
		{
			name: "image under /images/ kept with alt",
			in:   `<img src="/images/abc.png" alt="A cat">`,
			want: `<img src="/images/abc.png" alt="A cat"/>`,
		},
		{
			name: "missing-content placeholder kept",
			in:   `<div data-missing-kind="picture" data-missing-src="file:///x.png"></div>`,
			want: `<div data-missing-kind="picture" data-missing-src="file:///x.png"></div>`,
		},
		{
			name: "javascript link scheme dropped",
			in:   `<a href="javascript:alert(1)">click</a>`,
			want: `<a rel="noopener">click</a>`,
		},
		{
			name: "http, https and mailto links kept with rel=noopener",
			in:   `<a href="http://example.com">a</a><a href="https://example.com">b</a><a href="mailto:x@example.com">c</a>`,
			want: `<a href="http://example.com" rel="noopener">a</a><a href="https://example.com" rel="noopener">b</a><a href="mailto:x@example.com" rel="noopener">c</a>`,
		},
		{
			name: "table cell spans kept",
			in:   `<table><tr><td colspan="2" rowspan="1">x</td></tr></table>`,
			want: `<table><tbody><tr><td colspan="2" rowspan="1">x</td></tr></tbody></table>`,
		},
		{
			name: "h1 becomes h2, h4-h6 become h3",
			in:   `<h1>A</h1><h4>B</h4><h5>C</h5><h6>D</h6>`,
			want: `<h2>A</h2><h3>B</h3><h3>C</h3><h3>D</h3>`,
		},
		{
			name: "disallowed element unwrapped, allowed children kept",
			in:   `<section><p>kept</p></section>`,
			want: `<p>kept</p>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Sanitize(tt.in)
			if got != tt.want {
				t.Errorf("Sanitize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSanitizeIsIdempotent(t *testing.T) {
	inputs := []string{
		`<p onclick="x"><strong>Hi</strong></p>`,
		`<h1>Title</h1><ul><li>one</li><li>two</li></ul>`,
		`<a href="https://example.com" class="x">link</a>`,
		`<img src="/images/a.png" alt="a" style="width:1px">`,
	}
	for _, in := range inputs {
		once := Sanitize(in)
		twice := Sanitize(once)
		if once != twice {
			t.Errorf("Sanitize not idempotent for %q:\n first: %q\nsecond: %q", in, once, twice)
		}
	}
}
