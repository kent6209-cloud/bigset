package arcgis

import (
	"fmt"
	"net/http"

	"wms-server/internal/config"
)

type ServiceEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ServicesDir struct {
	CurrentVersion string         `json:"currentVersion"`
	Folders        []string       `json:"folders"`
	Services       []ServiceEntry `json:"services"`
}

func handleServices(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	dir := ServicesDir{
		CurrentVersion: CurrentVersion,
		Folders:        []string{},
	}
	for _, l := range cfg.Layers {
		dir.Services = append(dir.Services, ServiceEntry{Name: l.Name, Type: "MapServer"})
	}

	html := "<h2>ArcGIS Services</h2><ul>"
	for _, s := range dir.Services {
		html += fmt.Sprintf(`<li><a href="%s/MapServer">%s (MapServer)</a></li>`, s.Name, s.Name)
	}
	html += "</ul>"

	respond(w, r, dir, html)
}
