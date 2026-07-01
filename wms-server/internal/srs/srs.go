package srs

import "math"

const (
	EPSG4326 = "EPSG:4326"
	EPSG3857 = "EPSG:3857"

	EarthRadiusConst = 6378137.0
	earthRadius     = EarthRadiusConst
	originShift     = 2 * math.Pi * earthRadius / 2.0
)

func LonLatToMercator(lon, lat float64) (float64, float64) {
	x := lon * originShift / 180.0
	y := math.Log(math.Tan((90+lat)*math.Pi/360.0)) / (math.Pi / 180.0)
	y = y * originShift / 180.0
	return x, y
}

func MercatorToLonLat(x, y float64) (float64, float64) {
	lon := x / originShift * 180.0
	lat := y / originShift * 180.0
	lat = 180.0 / math.Pi * (2*math.Atan(math.Exp(lat*math.Pi/180.0)) - math.Pi/2.0)
	return lon, lat
}

func BBox4326To3857(minLon, minLat, maxLon, maxLat float64) (float64, float64, float64, float64) {
	x1, y1 := LonLatToMercator(minLon, minLat)
	x2, y2 := LonLatToMercator(maxLon, maxLat)
	return x1, y1, x2, y2
}

func TileBounds(z, x, y int) (minX, minY, maxX, maxY float64) {
	n := math.Pow(2, float64(z))
	minX = float64(x)/n*2*originShift - originShift
	maxX = float64(x+1)/n*2*originShift - originShift
	minY = -(float64(y)/n*2*originShift - originShift)
	maxY = -(float64(y+1)/n*2*originShift - originShift)
	if minY > maxY {
		minY, maxY = maxY, minY
	}
	return
}

func TileAtLonLat(lon, lat float64, z int) (x, y int) {
	n := math.Pow(2, float64(z))
	x = int(math.Floor((lon + 180.0) / 360.0 * n))
	y = int(math.Floor((1.0 - math.Log(math.Tan(lat*math.Pi/180.0)+1.0/math.Cos(lat*math.Pi/180.0))/math.Pi) / 2.0 * n))
	return
}

func EarthRadius() float64 { return EarthRadiusConst }

func BestZoom(resolution float64) int {
	initialRes := 2 * math.Pi * earthRadius / 256.0
	z := int(math.Round(math.Log2(initialRes / resolution)))
	if z < 0 {
		z = 0
	}
	if z > 22 {
		z = 22
	}
	return z
}
