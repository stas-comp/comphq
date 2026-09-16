package people

import (
	"net/http"
	"strconv"
	"time"
)

const cookieName = "comphq_person"
const cookieMaxAge = 400 * 24 * time.Hour // SPEC B4: Max-Age 400 days

// setPersonCookie sets (or refreshes) the person cookie. This is
// attribution only, not security (SPEC B4).
func setPersonCookie(w http.ResponseWriter, personID int64) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    strconv.FormatInt(personID, 10),
		Path:     "/",
		MaxAge:   int(cookieMaxAge.Seconds()),
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
}

// personIDFromCookie reads the person id from the request's cookie, if any.
func personIDFromCookie(r *http.Request) (int64, bool) {
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return 0, false
	}
	id, err := strconv.ParseInt(c.Value, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}
