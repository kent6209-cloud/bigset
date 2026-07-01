package arcgis

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const (
	CurrentVersion = "10.91"
	FullVersion    = "10.9.1"
)

type SpatialReference struct {
	WKID       int `json:"wkid"`
	LatestWKID int `json:"latestWkid"`
}

type Extent struct {
	XMin float64 `json:"xmin"`
	YMin float64 `json:"ymin"`
	XMax float64 `json:"xmax"`
	YMax float64 `json:"ymax"`
	SpatialReference `json:"spatialReference"`
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func writeHTML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html><html><head><meta charset="utf-8"><title>ArcGIS REST</title></head><body>`)
	fmt.Fprint(w, body)
	fmt.Fprint(w, `</body></html>`)
}

func respond(w http.ResponseWriter, r *http.Request, jsonData interface{}, htmlBody string) {
	f := strings.ToLower(r.URL.Query().Get("f"))
	switch f {
	case "json":
		writeJSON(w, jsonData)
	default:
		writeHTML(w, htmlBody)
	}
}
