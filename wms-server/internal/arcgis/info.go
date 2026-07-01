package arcgis

import (
	"net/http"
)

type ServerInfo struct {
	CurrentVersion string `json:"currentVersion"`
	FullVersion    string `json:"fullVersion"`
	AuthInfo       struct {
		IsTokenBasedSecurity bool `json:"isTokenBasedSecurity"`
	} `json:"authInfo"`
	OwningSystemURL string `json:"owningSystemUrl"`
}

func handleInfo(w http.ResponseWriter, r *http.Request) {
	info := ServerInfo{
		CurrentVersion: CurrentVersion,
		FullVersion:    FullVersion,
		OwningSystemURL: "",
	}
	info.AuthInfo.IsTokenBasedSecurity = false

	respond(w, r, info, `<h2>ArcGIS REST API</h2><p>Version: `+CurrentVersion+`</p><a href="services">Services</a>`)
}
