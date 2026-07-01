package arcgis

import (
	"net/http"
	"strconv"
	"strings"

	"wms-server/internal/config"
	"wms-server/internal/datasource"
)

func handleTile(w http.ResponseWriter, r *http.Request, layer config.LayerConfig, sources map[string]datasource.DataSource, tail string) {
	// tail = "z/y/x" or "z/y/x.jpg"
	tail = strings.TrimSuffix(tail, ".png")
	tail = strings.TrimSuffix(tail, ".jpg")
	tail = strings.TrimSuffix(tail, ".jpeg")

	parts := strings.Split(tail, "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid tile path", http.StatusBadRequest)
		return
	}

	z, err := strconv.Atoi(parts[0])
	if err != nil || z < 0 || z > 22 {
		http.Error(w, "Invalid z", http.StatusBadRequest)
		return
	}
	y, err := strconv.Atoi(parts[1])
	if err != nil || y < 0 {
		http.Error(w, "Invalid y", http.StatusBadRequest)
		return
	}
	x, err := strconv.Atoi(parts[2])
	if err != nil || x < 0 {
		http.Error(w, "Invalid x", http.StatusBadRequest)
		return
	}

	src, ok := sources[layer.Name]
	if !ok {
		http.Error(w, "Layer not found", http.StatusNotFound)
		return
	}

	data, format, err := src.GetTile(z, x, y)
	if err != nil {
		http.Error(w, "Tile not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", format)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}
