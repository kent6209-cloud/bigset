package wms

import (
	"fmt"
	"image/color"
	"math"
	"net/url"
	"strconv"
	"strings"

	"wms-server/internal/datasource"
	"wms-server/internal/renderer"
	"wms-server/internal/srs"
)

type GetMapRequest struct {
	Version     string
	Layers      []string
	Styles      []string
	CRS         string
	BBox        [4]float64
	Width       int
	Height      int
	Format      string
	Transparent bool
	BGColor     color.Color
}

func ParseGetMap(params url.Values) (*GetMapRequest, error) {
	req := &GetMapRequest{
		Version: params.Get("VERSION"),
		Format:  params.Get("FORMAT"),
	}

	layers := params.Get("LAYERS")
	if layers == "" {
		return nil, fmt.Errorf("缺少 LAYERS 參數")
	}
	req.Layers = strings.Split(layers, ",")

	crs := params.Get("CRS")
	if crs == "" {
		return nil, fmt.Errorf("缺少 CRS 參數")
	}
	req.CRS = strings.ToUpper(crs)

	bboxStr := params.Get("BBOX")
	if bboxStr == "" {
		return nil, fmt.Errorf("缺少 BBOX 參數")
	}
	parts := strings.Split(bboxStr, ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("BBOX 需要 4 個數值")
	}
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return nil, fmt.Errorf("BBOX 數值錯誤: %w", err)
		}
		req.BBox[i] = v
	}

	width, err := strconv.Atoi(params.Get("WIDTH"))
	if err != nil || width <= 0 || width > 4096 {
		return nil, fmt.Errorf("WIDTH 無效")
	}
	req.Width = width

	height, err := strconv.Atoi(params.Get("HEIGHT"))
	if err != nil || height <= 0 || height > 4096 {
		return nil, fmt.Errorf("HEIGHT 無效")
	}
	req.Height = height

	req.Transparent = strings.ToUpper(params.Get("TRANSPARENT")) == "TRUE"

	bgColor := params.Get("BGCOLOR")
	if bgColor != "" {
		bgColor = strings.TrimPrefix(bgColor, "0x")
		bgColor = strings.TrimPrefix(bgColor, "0X")
		if len(bgColor) == 6 {
			r, _ := strconv.ParseInt(bgColor[0:2], 16, 0)
			g, _ := strconv.ParseInt(bgColor[2:4], 16, 0)
			b, _ := strconv.ParseInt(bgColor[4:6], 16, 0)
			req.BGColor = color.RGBA{uint8(r), uint8(g), uint8(b), 255}
		}
	}

	if req.Format == "" {
		req.Format = "image/png"
	}
	return req, nil
}

func HandleGetMap(params url.Values, sources map[string]datasource.DataSource, _ []string) ([]byte, string, error) {
	req, err := ParseGetMap(params)
	if err != nil {
		return nil, "", fmt.Errorf("參數錯誤: %w", err)
	}

	rend := renderer.New(req.Width, req.Height, req.BBox, req.Transparent, req.BGColor)

	for _, layerName := range req.Layers {
		src, ok := sources[layerName]
		if !ok {
			continue
		}
		switch src.Type() {
		case "mbtiles", "xyz":
			renderTiles(rend, src, req)
		case "shapefile":
			renderFeatures(rend, src, req)
		}
	}

	data, err := rend.Image(req.Format)
	if err != nil {
		return nil, "", fmt.Errorf("渲染失敗: %w", err)
	}
	return data, req.Format, nil
}

func renderTiles(rend *renderer.Renderer, src datasource.DataSource, req *GetMapRequest) {
	bbox3857 := req.BBox
	if req.CRS == srs.EPSG4326 {
		x1, y1, x2, y2 := srs.BBox4326To3857(req.BBox[0], req.BBox[1], req.BBox[2], req.BBox[3])
		bbox3857 = [4]float64{x1, y1, x2, y2}
	}

	res := math.Abs(bbox3857[2]-bbox3857[0]) / float64(req.Width)
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

	spreadX := req.Width/256 + 2
	spreadY := req.Height/256 + 2

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
			if req.CRS == srs.EPSG4326 {
				lon1, lat1 := srs.MercatorToLonLat(tMinX, tMinY)
				lon2, lat2 := srs.MercatorToLonLat(tMaxX, tMaxY)
				tileBBox = [4]float64{lon1, lat1, lon2, lat2}
			}

			tileData, _, err := src.GetTile(zoom, tx, ty)
			if err != nil {
				continue
			}
			if err := rend.DrawTile(tileData, tileBBox); err != nil {
				continue
			}
		}
	}
}

func renderFeatures(rend *renderer.Renderer, src datasource.DataSource, req *GetMapRequest) {
	bbox := datasource.BBox{
		MinX: req.BBox[0],
		MinY: req.BBox[1],
		MaxX: req.BBox[2],
		MaxY: req.BBox[3],
	}
	features, err := src.GetFeatures(bbox, req.CRS)
	if err != nil {
		return
	}
	fill := color.RGBA{0, 120, 215, 180}
	stroke := color.RGBA{0, 60, 140, 255}
	for _, f := range features {
		rend.DrawFeature(f, fill, stroke, 1.5)
	}
}
