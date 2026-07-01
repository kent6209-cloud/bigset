package datasource

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type XYZSource struct {
	name string
	crs  string
	base string
	ext  string
}

func NewXYZSource(name, path, crs string) (*XYZSource, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("XYZ 目錄不存在: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("XYZ 路徑必須是目錄")
	}
	ext := "png"
	entries, _ := os.ReadDir(path)
	for _, e := range entries {
		if e.IsDir() {
			sub, _ := os.ReadDir(filepath.Join(path, e.Name()))
			for _, s := range sub {
				if !s.IsDir() {
					name := s.Name()
					if strings.HasSuffix(name, ".png") {
						ext = "png"
					} else if strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") {
						ext = "jpg"
					}
				}
			}
		}
	}
	return &XYZSource{name: name, crs: crs, base: path, ext: ext}, nil
}

func (x *XYZSource) Name() string { return x.name }
func (x *XYZSource) Type() string { return "xyz" }
func (x *XYZSource) CRS() string  { return x.crs }

func (x *XYZSource) GetTile(z, xT, y int) ([]byte, string, error) {
	tilePath := filepath.Join(x.base, fmt.Sprintf("%d", z), fmt.Sprintf("%d", xT), fmt.Sprintf("%d.%s", y, x.ext))
	data, err := os.ReadFile(tilePath)
	if err != nil {
		return nil, "", fmt.Errorf("讀取切片檔案 %s: %w", tilePath, err)
	}
	var format string
	switch x.ext {
	case "png":
		format = "image/png"
	case "jpg", "jpeg":
		format = "image/jpeg"
	default:
		format = "image/png"
	}
	return data, format, nil
}

func (x *XYZSource) GetFeatures(bbox BBox, targetCRS string) ([]Feature, error) {
	return nil, fmt.Errorf("XYZ 不支援向量查詢")
}
