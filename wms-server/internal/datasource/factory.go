package datasource

import (
	"fmt"

	"wms-server/internal/config"
)

func FromConfig(layer config.LayerConfig) (DataSource, error) {
	switch layer.Type {
	case "mbtiles":
		return NewMBTilesSource(layer.Name, layer.Path, layer.CRS)
	case "xyz":
		return NewXYZSource(layer.Name, layer.Path, layer.CRS)
	case "shapefile":
		return NewShapefileSource(layer.Name, layer.Path, layer.CRS)
	default:
		return nil, fmt.Errorf("不支援的資料源類型: %s", layer.Type)
	}
}
