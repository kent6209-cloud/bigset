package arcgis

import (
	"fmt"
	"image/color"
	"math"
	"net/http"
	"strconv"
	"strings"

	"wms-server/internal/config"
	"wms-server/internal/datasource"
	"wms-server/internal/renderer"
	"wms-server/internal/srs"
)

func handleExport(w http.ResponseWriter, r *http.Request, layer config.LayerConfig, sources map[string]datasource.DataSource) {
	q := r.URL.Query()

	bboxStr := q.Get("bbox")
	if bboxStr == "" {
		http.Error(w, "Missing bbox", http.StatusBadRequest)
		return
	}
	parts := strings.Split(bboxStr, ",")
	if len(parts) != 4 {
		http.Error(w, "Invalid bbox", http.StatusBadRequest)
		return
	}
	var bbox [4]float64
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			http.Error(w, "Invalid bbox value", http.StatusBadRequest)
			return
		}
		bbox[i] = v
	}

	sizeStr := q.Get("size")
	if sizeStr == "" {
		sizeStr = "800,600"
	}
	sizeParts := strings.Split(sizeStr, ",")
	width, _ := strconv.Atoi(strings.TrimSpace(sizeParts[0]))
	height := width
	if len(sizeParts) > 1 {
		height, _ = strconv.Atoi(strings.TrimSpace(sizeParts[1]))
	}
	if width <= 0 || height <= 0 {
		width, height = 800, 600
	}

	format := q.Get("format")
	switch strings.ToLower(format) {
	case "jpg", "jpeg":
		format = "image/jpeg"
	default:
		format = "image/png"
	}

	transparent := strings.ToLower(q.Get("transparent")) == "true"

	crs := layer.CRS
	bboxSR := q.Get("bboxSR")
	if bboxSR == "3857" || bboxSR == "102100" {
		crs = srs.EPSG3857
	} else if bboxSR == "4326" {
		crs = srs.EPSG4326
	}

	bboxLayer := bbox
	if crs != layer.CRS {
		if crs == srs.EPSG3857 && layer.CRS == srs.EPSG4326 {
			lon1, lat1 := srs.MercatorToLonLat(bbox[0], bbox[1])
			lon2, lat2 := srs.MercatorToLonLat(bbox[2], bbox[3])
			bboxLayer = [4]float64{lon1, lat1, lon2, lat2}
		} else if crs == srs.EPSG4326 && layer.CRS == srs.EPSG3857 {
			x1, y1 := srs.LonLatToMercator(bbox[0], bbox[1])
			x2, y2 := srs.LonLatToMercator(bbox[2], bbox[3])
			bboxLayer = [4]float64{x1, y1, x2, y2}
		}
	}

	rend := renderer.New(width, height, bboxLayer, transparent, nil)

	src, ok := sources[layer.Name]
	if !ok {
		http.Error(w, "Layer source not found", http.StatusNotFound)
		return
	}

	switch layer.Type {
	case "mbtiles", "xyz":
		renderTilesArcGIS(rend, src, bboxLayer, layer.CRS, width, height)
	case "shapefile":
		bboxDS := datasource.BBox{MinX: bboxLayer[0], MinY: bboxLayer[1], MaxX: bboxLayer[2], MaxY: bboxLayer[3]}
		features, err := src.GetFeatures(bboxDS, layer.CRS)
		if err == nil {
			fill := color.RGBA{0, 120, 215, 180}
			stroke := color.RGBA{0, 60, 140, 255}
			for _, f := range features {
				rend.DrawFeature(f, fill, stroke, 1.5)
			}
		}
	}

	data, err := rend.Image(format)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", format)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Write(data)
}

func renderTilesArcGIS(rend *renderer.Renderer, src datasource.DataSource, bbox [4]float64, crs string, width, height int) {
	bbox3857 := bbox
	if crs == srs.EPSG4326 {
		x1, y1, x2, y2 := srs.BBox4326To3857(bbox[0], bbox[1], bbox[2], bbox[3])
		bbox3857 = [4]float64{x1, y1, x2, y2}
	}
	res := math.Abs(bbox3857[2]-bbox3857[0]) / float64(width)
	zoom := srs.BestZoom(res)
	if zoom < 0 {
		zoom = 0
	}
	if zoom > 22 {
		zoom = 22
	}

	centerX := (bbox3857[0] + bbox3857[2]) / 2
	centerY := (bbox3857[1] + bbox3857[3]) / 2
	centerLon, centerLat := srs.MercatorToLonLat(centerX, centerY)
	cx, cy := srs.TileAtLonLat(centerLon, centerLat, zoom)

	spreadX := width/256 + 2
	spreadY := height/256 + 2

	for dy := -spreadY; dy <= spreadY; dy++ {
		for dx := -spreadX; dx <= spreadX; dx++ {
			tx, ty := cx+dx, cy+dy
			if tx < 0 || ty < 0 {
				continue
			}
			tMinX, tMinY, tMaxX, tMaxY := srs.TileBounds(zoom, tx, ty)
			if tMaxX < bbox3857[0] || tMinX > bbox3857[2] || tMaxY < bbox3857[1] || tMinY > bbox3857[3] {
				continue
			}
			tileBBox := [4]float64{tMinX, tMinY, tMaxX, tMaxY}
			if crs == srs.EPSG4326 {
				lon1, lat1 := srs.MercatorToLonLat(tMinX, tMinY)
				lon2, lat2 := srs.MercatorToLonLat(tMaxX, tMaxY)
				tileBBox = [4]float64{lon1, lat1, lon2, lat2}
			}
			tileData, _, err := src.GetTile(zoom, tx, ty)
			if err != nil {
				continue
			}
			rend.DrawTile(tileData, tileBBox)
		}
	}
}
