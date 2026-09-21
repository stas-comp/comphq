package settings

import (
	"io/fs"
	"net/http"

	"github.com/stas-comp/comphq"
)

type setupPageData struct {
	Current       string
	ChromeCommand string
	EdgeCommand   string
	BraveCommand  string
}

// BuildSetupCommands builds the desktop-shortcut commands (SPEC B6, B11.4)
// from the request's Host, for the three browsers the office uses. It is a
// pure function so the exact text is easy to test without a running server.
// Each one starts the browser with --app=, which shows Comp HQ in a window of
// its own with no address bar, tabs or bookmarks bar; they are the fallback
// route on the Setup page, and the only one for Brave.
func BuildSetupCommands(host string) (chrome, edge, brave string) {
	url := "http://" + host + "/"
	chrome = `"C:\Program Files\Google\Chrome\Application\chrome.exe" --app=` + url
	edge = `"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe" --app=` + url
	brave = `"C:\Program Files\BraveSoftware\Brave-Browser\Application\brave.exe" --app=` + url
	return chrome, edge, brave
}

func (h *Handlers) handleSetup(w http.ResponseWriter, r *http.Request) {
	chrome, edge, brave := BuildSetupCommands(r.Host)
	data := setupPageData{Current: "setup", ChromeCommand: chrome, EdgeCommand: edge, BraveCommand: brave}
	h.srv.RenderFrame(w, r, http.StatusOK, "settings-setup.html", "Set up this computer", data)
}

// downloadIconName is the name the saved icon file gets: what Windows' Change
// Icon dialog will show a person browsing for it, so it reads as Comp HQ's.
const downloadIconName = "Comp HQ.ico"

// handleSetupIcon serves the Comp HQ icon as a download (SPEC B11.4, gate
// 5.25) for the last-resort route on the Setup page: if Windows ever draws a
// shortcut with a blank icon, a person saves this and points the shortcut at
// it by hand.
func (h *Handlers) handleSetupIcon(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(comphq.Static, "web/static/theme/icons/comphq.ico")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/x-icon")
	w.Header().Set("Content-Disposition", `attachment; filename="`+downloadIconName+`"; filename*=UTF-8''Comp%20HQ.ico`)
	w.Write(data)
}
