package settings

import (
	"net/http"

	"github.com/stas-comp/comphq/internal/app/format"
	"github.com/stas-comp/comphq/internal/db"
)

type backupsPageData struct {
	Current         string
	HasRun          bool
	LastBackupAt    string
	LastBackupOK    bool
	LastBackupError string
	DailyKeep       int
}

func (h *Handlers) handleBackups(w http.ResponseWriter, r *http.Request) {
	status, err := db.GetBackupStatus(h.srv.DB)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	data := backupsPageData{
		Current:         "backups",
		HasRun:          status.HasRun,
		LastBackupOK:    status.LastBackupOK,
		LastBackupError: status.LastBackupError,
		DailyKeep:       db.DailyBackupsToKeep,
	}
	if status.HasRun {
		data.LastBackupAt = format.DateTime(status.LastBackupAt.Local())
	}

	h.srv.RenderFrame(w, r, http.StatusOK, "settings-backups.html", "Backups", data)
}
