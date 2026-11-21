package brand

import "testing"

func TestLookup(t *testing.T) {
	for key, want := range map[string]string{"": "Comp HQ", "comp": "Comp HQ", "mc": "MC HQ", " MC ": "MC HQ"} {
		b, err := Lookup(key)
		if err != nil {
			t.Errorf("Lookup(%q): %v", key, err)
			continue
		}
		if b.Name != want {
			t.Errorf("Lookup(%q).Name = %q, want %q", key, b.Name, want)
		}
	}
	if _, err := Lookup("mchq"); err == nil {
		t.Error(`Lookup("mchq") succeeded; an unknown key must be an error`)
	}
}

func TestSetAndCurrent(t *testing.T) {
	t.Cleanup(func() { current = Comp })
	if Current().Key != "comp" {
		t.Fatalf("the default brand is %q, want comp", Current().Key)
	}
	if err := Set("mc"); err != nil {
		t.Fatal(err)
	}
	if Current().Name != "MC HQ" {
		t.Errorf("after Set(mc), Current().Name = %q", Current().Name)
	}
	if err := Set("nope"); err == nil {
		t.Error("Set(nope) succeeded")
	}
	if Current().Name != "MC HQ" {
		t.Error("a failed Set changed the brand")
	}
}

func TestIconDownload(t *testing.T) {
	if got, want := Comp.IconDownloadDisposition(), `attachment; filename="Comp HQ.ico"; filename*=UTF-8''Comp%20HQ.ico`; got != want {
		t.Errorf("Comp disposition = %s, want %s", got, want)
	}
	if got, want := MC.IconPath(), "web/static/theme/icons/mc/mchq.ico"; got != want {
		t.Errorf("MC.IconPath() = %s, want %s", got, want)
	}
	if got, want := Comp.IconPath(), "web/static/theme/icons/comphq.ico"; got != want {
		t.Errorf("Comp.IconPath() = %s, want %s", got, want)
	}
}
