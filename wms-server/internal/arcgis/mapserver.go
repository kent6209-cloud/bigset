package arcgis

import (
	"fmt"
	"net/http"
	"strings"

	"wms-server/internal/config"
	"wms-server/internal/srs"
)

type LayerInfo struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	ParentLayerID    int    `json:"parentLayerId"`
	DefaultVisibility bool  `json:"defaultVisibility"`
	SubLayerIDs      interface{} `json:"subLayerIds"`
	MinScale         int    `json:"minScale"`
	MaxScale         int    `json:"maxScale"`
}

type MapServerInfo struct {
	CurrentVersion          string           `json:"currentVersion"`
	ServiceDescription     string           `json:"serviceDescription"`
	MapName                string           `json:"mapName"`
	Description            string           `json:"description"`
	CopyrightText          string           `json:"copyrightText"`
	SupportsDynamicLayers  bool             `json:"supportsDynamicLayers"`
	Layers                 []LayerInfo      `json:"layers"`
	Tables                 []string         `json:"tables"`
	SpatialReference       SpatialReference `json:"spatialReference"`
	SingleFusedMapCache    bool             `json:"singleFusedMapCache"`
	InitialExtent          Extent           `json:"initialExtent"`
	FullExtent             Extent           `json:"fullExtent"`
	Units                  string           `json:"units"`
	SupportedImageFormatTypes string         `json:"supportedImageFormatTypes"`
	Capabilities           string           `json:"capabilities"`
	ExportTilesAllowed     bool             `json:"exportTilesAllowed"`
	MaxRecordCount         int              `json:"maxRecordCount"`
	MaxImageHeight         int              `json:"maxImageHeight"`
	MaxImageWidth          int              `json:"maxImageWidth"`
	SupportedExtensions    string           `json:"supportedExtensions"`
}

func mapUnits(crs string) string {
	if crs == srs.EPSG4326 {
		return "esriDecimalDegrees"
	}
	return "esriMeters"
}

func extentFromCRS(crs string) Extent {
	wkid := 4326
	var xmin, ymin, xmax, ymax float64
	if crs == srs.EPSG3857 {
		wkid = 3857
		xmin = -20037508.34
		ymin = -20037508.34
		xmax = 20037508.34
		ymax = 20037508.34
	} else {
		xmin = -180
		ymin = -90
		xmax = 180
		ymax = 90
	}
	return Extent{
		XMin: xmin, YMin: ymin, XMax: xmax, YMax: ymax,
		SpatialReference: SpatialReference{WKID: wkid, LatestWKID: wkid},
	}
}

func handleMapServer(w http.ResponseWriter, r *http.Request, layer config.LayerConfig) {
	wkid := 4326
	if layer.CRS == srs.EPSG3857 {
		wkid = 3857
	}
	info := MapServerInfo{
		CurrentVersion:         CurrentVersion,
		ServiceDescription:     layer.Abstract,
		MapName:                layer.Name,
		Description:            layer.Abstract,
		CopyrightText:          "",
		SupportsDynamicLayers:  false,
		Layers: []LayerInfo{
			{ID: 0, Name: layer.Title, ParentLayerID: -1, DefaultVisibility: true, SubLayerIDs: nil, MinScale: 0, MaxScale: 0},
		},
		Tables:                 []string{},
		SpatialReference:       SpatialReference{WKID: wkid, LatestWKID: wkid},
		SingleFusedMapCache:    layer.Type == "mbtiles" || layer.Type == "xyz",
		InitialExtent:          extentFromCRS(layer.CRS),
		FullExtent:             extentFromCRS(layer.CRS),
		Units:                  mapUnits(layer.CRS),
		SupportedImageFormatTypes: "PNG,JPEG",
		Capabilities:           "Map,Query,TilesOnly",
		ExportTilesAllowed:     true,
		MaxRecordCount:         1000,
		MaxImageHeight:         4096,
		MaxImageWidth:          4096,
		SupportedExtensions:    "",
	}

	html := fmt.Sprintf(`<h2>%s (MapServer)</h2>
<p>%s</p>
<ul>
<li><a href="%s/export">Export Map</a></li>
<li><a href="%s/identify">Identify</a></li>
</ul>`, layer.Title, layer.Abstract, layer.Name, layer.Name)

	respond(w, r, info, html)
}

func parseServicePath(path string) (serviceName string, endpoint string) {
	// path like: /arcgis/rest/services/{name}/MapServer/export
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, p := range parts {
		if p == "services" && i+1 < len(parts) {
			serviceName = parts[i+1]
			if i+2 < len(parts) && parts[i+2] == "MapServer" {
				if i+3 < len(parts) {
					endpoint = strings.Join(parts[i+3:], "/")
				} else {
					endpoint = ""
				}
			}
			break
		}
	}
	return
}
