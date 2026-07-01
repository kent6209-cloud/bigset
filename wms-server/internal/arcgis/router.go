package arcgis

import (
	"net/http"
	"strings"

	"wms-server/internal/config"
	"wms-server/internal/datasource"
)

type ArcGISHandler struct {
	cfg     *config.Config
	sources map[string]datasource.DataSource
}

func NewHandler(cfg *config.Config, sources map[string]datasource.DataSource) *ArcGISHandler {
	return &ArcGISHandler{cfg: cfg, sources: sources}
}

func (h *ArcGISHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")

	// Expected paths:
	// arcgis/rest/info
	// arcgis/rest/services
	// arcgis/rest/services/{name}/MapServer
	// arcgis/rest/services/{name}/MapServer/export
	// arcgis/rest/services/{name}/MapServer/identify
	// arcgis/rest/services/{name}/MapServer/tile/{z}/{y}/{x}

	if len(parts) < 3 || parts[0] != "arcgis" || parts[1] != "rest" {
		http.NotFound(w, r)
		return
	}

	switch {
	case len(parts) == 3 && parts[2] == "info":
		handleInfo(w, r)
	case len(parts) == 3 && parts[2] == "services":
		handleServices(w, r, h.cfg)
	case len(parts) >= 5 && parts[3] != "" && parts[4] == "MapServer":
		serviceName := parts[3]
		layer := findLayer(h.cfg, serviceName)
		if layer == nil {
			http.NotFound(w, r)
			return
		}
		if len(parts) == 5 {
			handleMapServer(w, r, *layer)
		} else if len(parts) >= 6 {
			endpoint := strings.Join(parts[5:], "/")
			switch {
			case endpoint == "export":
				handleExport(w, r, *layer, h.sources)
			case endpoint == "identify":
				handleIdentify(w, r, *layer, h.sources)
			case strings.HasPrefix(endpoint, "tile/"):
				tail := strings.TrimPrefix(endpoint, "tile/")
				handleTile(w, r, *layer, h.sources, tail)
			default:
				http.NotFound(w, r)
			}
		}
	default:
		http.NotFound(w, r)
	}
}

func findLayer(cfg *config.Config, name string) *config.LayerConfig {
	for _, l := range cfg.Layers {
		if l.Name == name {
			return &l
		}
	}
	return nil
}
