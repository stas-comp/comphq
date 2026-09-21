package settings

import "testing"

func TestBuildSetupCommands(t *testing.T) {
	chrome, edge, brave := BuildSetupCommands("192.168.1.50:8080")

	wantChrome := `"C:\Program Files\Google\Chrome\Application\chrome.exe" --app=http://192.168.1.50:8080/`
	wantEdge := `"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe" --app=http://192.168.1.50:8080/`
	wantBrave := `"C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe" --app=http://192.168.1.50:8080/`

	if chrome != wantChrome {
		t.Errorf("chrome = %q, want %q", chrome, wantChrome)
	}
	if edge != wantEdge {
		t.Errorf("edge = %q, want %q", edge, wantEdge)
	}
	if brave != wantBrave {
		t.Errorf("brave = %q, want %q", brave, wantBrave)
	}
}

// Whatever address the page was reached at is the address in the command,
// so a shortcut made on any office computer opens the same Comp HQ.
func TestBuildSetupCommandsUseTheHostTheyAreGiven(t *testing.T) {
	for _, host := range []string{"nas.local:8080", "10.0.0.7:9000", "localhost:8090"} {
		chrome, edge, brave := BuildSetupCommands(host)
		for name, cmd := range map[string]string{"chrome": chrome, "edge": edge, "brave": brave} {
			want := " --app=http://" + host + "/"
			if len(cmd) < len(want) || cmd[len(cmd)-len(want):] != want {
				t.Errorf("%s command for %s = %q, want it to end %q", name, host, cmd, want)
			}
		}
	}
}
