package settings

import "net/http"

type aboutPageData struct {
	Current string
	Version string
}

func (h *Handlers) handleAbout(w http.ResponseWriter, r *http.Request) {
	data := aboutPageData{Current: "about", Version: h.srv.Version}
	h.srv.RenderFrame(w, r, http.StatusOK, "settings-about.html", "About", data)
}
