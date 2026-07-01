package arcgis

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"wms-server/internal/config"
	"wms-server/internal/datasource"
	"wms-server/internal/srs"
)

type IdentifyResult struct {
	LayerID   int                    `json:"layerId"`
	LayerName string                `json:"layerName"`
	Value     string                `json:"value"`
	DisplayFieldName string         `json:"displayFieldName"`
	Attributes map[string]string    `json:"attributes"`
	GeometryType string             `json:"geometryType"`
	Geometry    map[string]float64  `json:"geometry"`
}

type IdentifyResponse struct {
	Results []IdentifyResult `json:"results"`
}

func handleIdentify(w http.ResponseWriter, r *http.Request, layer config.LayerConfig, sources map[string]datasource.DataSource) {
	q := r.URL.Query()

	geometryStr := q.Get("geometry")
	if geometryStr == "" {
		http.Error(w, "Missing geometry", http.StatusBadRequest)
		return
	}
	parts := strings.Split(geometryStr, ",")
	if len(parts) < 2 {
		http.Error(w, "Invalid geometry", http.StatusBadRequest)
		return
	}
	px, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	py := px
	if len(parts) > 1 {
		py, _ = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	}

	tolerance, _ := strconv.ParseFloat(q.Get("tolerance"), 64)
	if tolerance <= 0 {
		tolerance = 0.01
	}

	mapExtent := q.Get("mapExtent")
	var mapBBox [4]float64
	if mapExtent != "" {
		extParts := strings.Split(mapExtent, ",")
		if len(extParts) == 4 {
			for i, p := range extParts {
				mapBBox[i], _ = strconv.ParseFloat(strings.TrimSpace(p), 64)
			}
		}
	}

	crs := layer.CRS
	sr := q.Get("sr")
	if sr == "3857" || sr == "102100" {
		crs = srs.EPSG3857
	} else if sr == "4326" {
		crs = srs.EPSG4326
	}

	// Convert point to layer CRS
	pointX, pointY := px, py
	if crs != layer.CRS {
		if crs == srs.EPSG3857 && layer.CRS == srs.EPSG4326 {
			pointX, pointY = srs.MercatorToLonLat(px, py)
		} else if crs == srs.EPSG4326 && layer.CRS == srs.EPSG3857 {
			pointX, pointY = srs.LonLatToMercator(px, py)
		}
	}

	searchBBox := datasource.BBox{
		MinX: pointX - tolerance,
		MinY: pointY - tolerance,
		MaxX: pointX + tolerance,
		MaxY: pointY + tolerance,
	}

	src, ok := sources[layer.Name]
	if !ok {
		http.Error(w, "Layer not found", http.StatusNotFound)
		return
	}

	features, err := src.GetFeatures(searchBBox, layer.CRS)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	results := []IdentifyResult{}
	for _, f := range features {
		geomX, geomY := pointX, pointY
		if len(f.Geometry) > 0 {
			geomX, geomY = f.Geometry[0].X, f.Geometry[0].Y
		}
		results = append(results, IdentifyResult{
			LayerID:         0,
			LayerName:       layer.Title,
			Value:           fmt.Sprintf("%d", f.ID),
			DisplayFieldName: "FID",
			Attributes:      f.Props,
			GeometryType:    "esriGeometryPoint",
			Geometry:        map[string]float64{"x": geomX, "y": geomY},
		})
	}

	if results == nil {
		results = []IdentifyResult{}
	}

	resp := IdentifyResponse{Results: results}
	respond(w, r, resp, fmt.Sprintf("<h2>Identify Results</h2><p>%d features found</p>", len(results)))
}
