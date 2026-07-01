package datasource

import (
	"fmt"
	"log"

	"wms-server/internal/config"
)

func FromConfig(layer config.LayerConfig) (DataSource, error) {
	var src DataSource
	var err error

	switch layer.Type {
	case "mbtiles":
		src, err = NewMBTilesSource(layer.Name, layer.Path, layer.CRS)
	case "xyz":
		src, err = NewXYZSource(layer.Name, layer.Path, layer.CRS)
	case "shapefile":
		src, err = NewShapefileSource(layer.Name, layer.Path, layer.CRS)
	case "pmtiles":
		src, err = NewPMTilesSource(layer.Name, layer.Path, layer.CRS)
	default:
		return nil, fmt.Errorf("不支援的資料源類型: %s", layer.Type)
	}
	if err != nil {
		return nil, err
	}

	// Wrap with LRU cache for tile-based sources
	if layer.Type == "mbtiles" || layer.Type == "xyz" || layer.Type == "pmtiles" {
		cached := NewCacheSource(src, 512)
		log.Printf("  ✓ 已啟用 LRU 快取 (512 tiles) 於 %s", layer.Name)
		return cached, nil
	}

	return src, nil
}
