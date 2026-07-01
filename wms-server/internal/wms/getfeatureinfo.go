package wms

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"wms-server/internal/datasource"
	"wms-server/internal/srs"
)

type GetFeatureInfoRequest struct {
	GetMapRequest
	QueryLayers []string
	InfoFormat  string
	I, J        int
}

func ParseGetFeatureInfo(params url.Values) (*GetFeatureInfoRequest, error) {
	req := &GetFeatureInfoRequest{}
	queryLayers := params.Get("QUERY_LAYERS")
	if queryLayers == "" {
		return nil, fmt.Errorf("缺少 QUERY_LAYERS 參數")
	}
	req.QueryLayers = strings.Split(queryLayers, ",")

	infoFormat := params.Get("INFO_FORMAT")
	if infoFormat == "" {
		infoFormat = "text/plain"
	}
	req.InfoFormat = infoFormat

	i, err := strconv.Atoi(params.Get("I"))
	if err != nil {
		return nil, fmt.Errorf("缺少或無效的 I 參數: %w", err)
	}
	req.I = i

	j, err := strconv.Atoi(params.Get("J"))
	if err != nil {
		return nil, fmt.Errorf("缺少或無效的 J 參數: %w", err)
	}
	req.J = j

	gmParams := url.Values{}
	for _, k := range []string{"VERSION", "LAYERS", "STYLES", "CRS", "BBOX", "WIDTH", "HEIGHT"} {
		if v := params.Get(k); v != "" {
			gmParams[k] = []string{v}
		}
	}
	gmReq, err := ParseGetMap(gmParams)
	if err != nil {
		return nil, fmt.Errorf("GetMap 參數解析失敗: %w", err)
	}
	req.GetMapRequest = *gmReq
	return req, nil
}

func HandleGetFeatureInfo(params url.Values, sources map[string]datasource.DataSource) (string, error) {
	req, err := ParseGetFeatureInfo(params)
	if err != nil {
		return "", err
	}

	// Convert pixel I,J to world coordinates
	lx := float64(req.I)/float64(req.Width)*(req.BBox[2]-req.BBox[0]) + req.BBox[0]
	ly := float64(req.J)/float64(req.Height)*(req.BBox[1]-req.BBox[3]) + req.BBox[3]

	// Convert to WGS84 for search
	queryLon, queryLat := lx, ly
	if req.CRS == srs.EPSG3857 {
		queryLon, queryLat = srs.MercatorToLonLat(lx, ly)
	}

	var results []string
	for _, layerName := range req.QueryLayers {
		src, ok := sources[layerName]
		if !ok {
			continue
		}
		searchBBox := datasource.BBox{
			MinX: queryLon - 0.001,
			MinY: queryLat - 0.001,
			MaxX: queryLon + 0.001,
			MaxY: queryLat + 0.001,
		}
		features, err := src.GetFeatures(searchBBox, srs.EPSG4326)
		if err != nil {
			continue
		}
		for _, f := range features {
			props := ""
			for k, v := range f.Props {
				props += fmt.Sprintf("  %s: %s\n", k, v)
			}
			results = append(results, fmt.Sprintf("Layer: %s\nFeature ID: %d\n%s", layerName, f.ID, props))
		}
	}

	if len(results) == 0 {
		return "No features found at the queried location.\n", nil
	}
	return strings.Join(results, "\n---\n"), nil
}
