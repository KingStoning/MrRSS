package settings

import (
	"MrRSS/internal/utils/fileutil"
	"encoding/json"
	"net/http"
)

// HandleFonts enumerates the desktop's installed font families, never the remote server's fonts.
func HandleFonts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if fileutil.IsServerMode() {
		http.Error(w, "desktop only", http.StatusNotImplemented)
		return
	}
	fonts, err := installedFontFamilies()
	if err != nil {
		http.Error(w, "font enumeration unavailable", http.StatusNotImplemented)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(fonts)
}
