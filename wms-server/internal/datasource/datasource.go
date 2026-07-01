package datasource

type GeometryType int

const (
	PointType      GeometryType = iota
	LineStringType
	PolygonType
)

type Coordinate struct {
	X, Y float64
}

type Feature struct {
	ID       int64
	Type     GeometryType
	Geometry []Coordinate
	Props    map[string]string
}

type BBox struct {
	MinX, MinY, MaxX, MaxY float64
}

type DataSource interface {
	Name() string
	Type() string
	CRS() string
	GetTile(z, x, y int) ([]byte, string, error)
	GetFeatures(bbox BBox, targetCRS string) ([]Feature, error)
}
