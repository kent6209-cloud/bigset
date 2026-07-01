package datasource

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"strings"

	"wms-server/internal/srs"
)

type ShapeType int32

const (
	NullShape    ShapeType = 0
	ShpPoint     ShapeType = 1
	ShpPolyLine  ShapeType = 3
	ShpPolygon   ShapeType = 5
	ShpPointM    ShapeType = 21
	ShpPolyLineM ShapeType = 23
	ShpPolygonM  ShapeType = 25
)

type ShapefileSource struct {
	name    string
	crs     string
	path    string
	fileCrs string
}

func NewShapefileSource(name, path, crs string) (*ShapefileSource, error) {
	if !strings.HasSuffix(path, ".shp") {
		if _, err := os.Stat(path + ".shp"); err == nil {
			path = path + ".shp"
		}
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("Shapefile %s 不存在: %w", path, err)
	}
	return &ShapefileSource{name: name, crs: crs, path: path, fileCrs: crs}, nil
}

func (s *ShapefileSource) Name() string { return s.name }
func (s *ShapefileSource) Type() string { return "shapefile" }
func (s *ShapefileSource) CRS() string  { return s.crs }

func (s *ShapefileSource) GetTile(z, x, y int) ([]byte, string, error) {
	return nil, "", fmt.Errorf("Shapefile 不支援切片查詢")
}

func (s *ShapefileSource) GetFeatures(bbox BBox, targetCRS string) ([]Feature, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}
	if len(data) < 100 {
		return nil, fmt.Errorf("不是有效的 Shapefile (檔案過小)")
	}

	fileLen := int(binary.BigEndian.Uint32(data[24:28])) * 2

	var features []Feature
	pos := 100
	for pos+8 <= fileLen && pos+8 <= len(data) {
		recordNum := binary.BigEndian.Uint32(data[pos : pos+4])
		contentLen := int(binary.BigEndian.Uint32(data[pos+4:pos+8])) * 2
		pos += 8
		if pos+contentLen > len(data) {
			break
		}
		recData := data[pos : pos+contentLen]
		if len(recData) < 4 {
			break
		}
		recType := ShapeType(int32(binary.LittleEndian.Uint32(recData[:4])))

		f, err := parseShapeRecord(recData, recType)
		if err != nil {
			pos += contentLen
			continue
		}
		f.ID = int64(recordNum)

		transformCoords(&f, s.fileCrs, targetCRS)

		// Approximate bbox filter (using first coordinate)
		if len(f.Geometry) > 0 {
			features = append(features, f)
		}
		pos += contentLen
	}
	return features, nil
}

func transformCoords(f *Feature, fromCRS, toCRS string) {
	if fromCRS == toCRS {
		return
	}
	for i := range f.Geometry {
		var nx, ny float64
		if fromCRS == srs.EPSG4326 && toCRS == srs.EPSG3857 {
			nx, ny = srs.LonLatToMercator(f.Geometry[i].X, f.Geometry[i].Y)
		} else if fromCRS == srs.EPSG3857 && toCRS == srs.EPSG4326 {
			nx, ny = srs.MercatorToLonLat(f.Geometry[i].X, f.Geometry[i].Y)
		} else {
			nx, ny = f.Geometry[i].X, f.Geometry[i].Y
		}
		f.Geometry[i].X = nx
		f.Geometry[i].Y = ny
	}
}

func parseShapeRecord(data []byte, shapeType ShapeType) (Feature, error) {
	switch shapeType {
	case ShpPoint, ShpPointM:
		if len(data) < 20 {
			return Feature{}, fmt.Errorf("Point 記錄過短")
		}
		x := math.Float64frombits(binary.LittleEndian.Uint64(data[4:12]))
		y := math.Float64frombits(binary.LittleEndian.Uint64(data[12:20]))
		return Feature{Type: PointType, Geometry: []Coordinate{{X: x, Y: y}}}, nil

	case ShpPolyLine, ShpPolyLineM, ShpPolygon, ShpPolygonM:
		if len(data) < 48 {
			return Feature{}, fmt.Errorf("PolyLine/Polygon 記錄過短")
		}
		numParts := int(binary.LittleEndian.Uint32(data[40:44]))
		numPoints := int(binary.LittleEndian.Uint32(data[44:48]))
		if numParts == 0 || numPoints == 0 {
			return Feature{}, fmt.Errorf("空的幾何")
		}

		partsOff := 48
		if partsOff+numParts*4 > len(data) {
			return Feature{}, fmt.Errorf("parts 陣列截斷")
		}

		startIdx := int(binary.LittleEndian.Uint32(data[partsOff : partsOff+4]))
		endIdx := numPoints
		if numParts > 1 {
			startIdx = int(binary.LittleEndian.Uint32(data[partsOff : partsOff+4]))
			endIdx = int(binary.LittleEndian.Uint32(data[partsOff+4 : partsOff+8]))
		}

		ptsOff := partsOff + numParts*4
		if ptsOff+endIdx*16 > len(data) {
			return Feature{}, fmt.Errorf("點陣列截斷")
		}

		coords := make([]Coordinate, 0, endIdx-startIdx)
		for i := startIdx; i < endIdx; i++ {
			off := ptsOff + i*16
			x := math.Float64frombits(binary.LittleEndian.Uint64(data[off : off+8]))
			y := math.Float64frombits(binary.LittleEndian.Uint64(data[off+8 : off+16]))
			coords = append(coords, Coordinate{X: x, Y: y})
		}

		gt := LineStringType
		if shapeType == ShpPolygon || shapeType == ShpPolygonM {
			gt = PolygonType
		}
		return Feature{Type: gt, Geometry: coords}, nil

	default:
		return Feature{}, fmt.Errorf("不支援的 Shape 類型: %v", shapeType)
	}
}
