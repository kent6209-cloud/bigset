package datasource

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

type MBTilesSource struct {
	name string
	crs  string
	db   *sql.DB
}

func NewMBTilesSource(name, path, crs string) (*MBTilesSource, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("MBTiles 檔案不存在: %w", err)
	}
	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?mode=ro", path))
	if err != nil {
		return nil, fmt.Errorf("無法開啟 MBTiles: %w", err)
	}
	var format string
	err = db.QueryRow("SELECT value FROM metadata WHERE name='format'").Scan(&format)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("無效的 MBTiles 檔案 (缺少 format): %w", err)
	}
	return &MBTilesSource{name: name, crs: crs, db: db}, nil
}

func (m *MBTilesSource) Name() string { return m.name }
func (m *MBTilesSource) Type() string { return "mbtiles" }
func (m *MBTilesSource) CRS() string  { return m.crs }

func (m *MBTilesSource) GetTile(z, x, y int) ([]byte, string, error) {
	tmsY := (1 << z) - 1 - y
	var data []byte
	err := m.db.QueryRow(
		"SELECT tile_data FROM tiles WHERE zoom_level=? AND tile_column=? AND tile_row=?",
		z, x, tmsY,
	).Scan(&data)
	if err != nil {
		return nil, "", fmt.Errorf("讀取切片 (%d/%d/%d): %w", z, x, y, err)
	}
	format := "image/png"
	if len(data) > 2 && data[0] == 0xFF && data[1] == 0xD8 {
		format = "image/jpeg"
	}
	return data, format, nil
}

func (m *MBTilesSource) GetFeatures(bbox BBox, targetCRS string) ([]Feature, error) {
	return nil, fmt.Errorf("MBTiles 不支援向量查詢")
}

func (m *MBTilesSource) Close() error {
	return m.db.Close()
}
