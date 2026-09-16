package settings

import "net/http"

type setupPageData struct {
	Current       string
	ChromeCommand string
	BraveCommand  string
}

// BuildSetupCommands builds the SPEC B6 desktop-shortcut commands from the
// request's Host, for both supported browsers. A pure function so its
// exact text is easy to test without a running server.
func BuildSetupCommands(host string) (chrome, brave string) {
	url := "http://" + host + "/"
	chrome = `"C:\Program Files\Google\Chrome\Application\chrome.exe" --app=` + url
	brave = `"C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe" --app=` + url
	return chrome, brave
}

func (h *Handlers) handleSetup(w http.ResponseWriter, r *http.Request) {
	chrome, brave := BuildSetupCommands(r.Host)
	data := setupPageData{Current: "setup", ChromeCommand: chrome, BraveCommand: brave}
	h.srv.RenderFrame(w, r, http.StatusOK, "settings-setup.html", "Set up this computer", data)
}
