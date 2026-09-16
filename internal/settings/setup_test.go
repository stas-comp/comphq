package settings

import "testing"

func TestBuildSetupCommands(t *testing.T) {
	chrome, brave := BuildSetupCommands("192.168.1.50:8080")

	wantChrome := `"C:\Program Files\Google\Chrome\Application\chrome.exe" --app=http://192.168.1.50:8080/`
	wantBrave := `"C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe" --app=http://192.168.1.50:8080/`

	if chrome != wantChrome {
		t.Errorf("chrome = %q, want %q", chrome, wantChrome)
	}
	if brave != wantBrave {
		t.Errorf("brave = %q, want %q", brave, wantBrave)
	}
}
