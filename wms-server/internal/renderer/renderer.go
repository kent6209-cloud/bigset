package renderer

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"

	"wms-server/internal/datasource"
)

type Renderer struct {
	width, height int
	bbox          [4]float64
	canvas        *image.RGBA
}

func New(width, height int, bbox [4]float64, transparent bool, bgColor color.Color) *Renderer {
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	if transparent {
		draw.Draw(canvas, canvas.Bounds(), image.Transparent, image.Point{}, draw.Src)
	} else {
		if bgColor == nil {
			bgColor = color.White
		}
		draw.Draw(canvas, canvas.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)
	}
	return &Renderer{width: width, height: height, bbox: bbox, canvas: canvas}
}

func (r *Renderer) worldToPixel(wx, wy float64) (int, int) {
	px := (wx - r.bbox[0]) / (r.bbox[2] - r.bbox[0]) * float64(r.width)
	py := (r.bbox[3] - wy) / (r.bbox[3] - r.bbox[1]) * float64(r.height)
	return int(math.Round(px)), int(math.Round(py))
}

func (r *Renderer) DrawTile(tileData []byte, tileBBox [4]float64) error {
	srcImg, _, err := image.Decode(bytes.NewReader(tileData))
	if err != nil {
		return fmt.Errorf("解碼切片失敗: %w", err)
	}

	sx := (tileBBox[0] - r.bbox[0]) / (r.bbox[2] - r.bbox[0]) * float64(r.width)
	sy := (r.bbox[3] - tileBBox[3]) / (r.bbox[3] - r.bbox[1]) * float64(r.height)
	ex := (tileBBox[2] - r.bbox[0]) / (r.bbox[2] - r.bbox[0]) * float64(r.width)
	ey := (r.bbox[3] - tileBBox[1]) / (r.bbox[3] - r.bbox[1]) * float64(r.height)

	dstW := int(math.Round(ex - sx))
	dstH := int(math.Round(ey - sy))
	if dstW <= 0 || dstH <= 0 {
		return nil
	}

	scaled := scaleImage(srcImg, dstW, dstH)
	draw.Draw(r.canvas, image.Rect(int(sx), int(sy), int(sx)+dstW, int(sy)+dstH), scaled, image.Point{}, draw.Over)
	return nil
}

func scaleImage(src image.Image, dstW, dstH int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	if srcW == 0 || srcH == 0 {
		return dst
	}

	for dy := 0; dy < dstH; dy++ {
		for dx := 0; dx < dstW; dx++ {
			sx := dx * srcW / dstW
			sy := dy * srcH / dstH
			dst.Set(dx, dy, src.At(sx+srcBounds.Min.X, sy+srcBounds.Min.Y))
		}
	}
	return dst
}

func (r *Renderer) DrawFeature(f datasource.Feature, fill, stroke color.Color, lw float64) {
	if len(f.Geometry) == 0 {
		return
	}
	switch f.Type {
	case datasource.PointType:
		px, py := r.worldToPixel(f.Geometry[0].X, f.Geometry[0].Y)
		radius := int(math.Max(lw, 3))
		r.drawFilledCircle(px, py, radius, fill)
	case datasource.LineStringType:
		r.drawPolyLine(f.Geometry, stroke, lw)
	case datasource.PolygonType:
		r.fillPolygon(f.Geometry, fill)
		r.drawPolyLine(f.Geometry, stroke, lw)
	}
}

func (r *Renderer) drawFilledCircle(cx, cy, radius int, c color.Color) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				x, y := cx+dx, cy+dy
				if x >= 0 && x < r.width && y >= 0 && y < r.height {
					r.canvas.Set(x, y, c)
				}
			}
		}
	}
}

func (r *Renderer) drawPolyLine(coords []datasource.Coordinate, c color.Color, _ float64) {
	for i := 0; i < len(coords)-1; i++ {
		x1, y1 := r.worldToPixel(coords[i].X, coords[i].Y)
		x2, y2 := r.worldToPixel(coords[i+1].X, coords[i+1].Y)
		r.drawLine(x1, y1, x2, y2, c)
	}
}

func (r *Renderer) drawLine(x1, y1, x2, y2 int, c color.Color) {
	dx := x2 - x1
	dy := y2 - y1
	steps := abs(dx)
	if abs(dy) > steps {
		steps = abs(dy)
	}
	if steps == 0 {
		if x1 >= 0 && x1 < r.width && y1 >= 0 && y1 < r.height {
			r.canvas.Set(x1, y1, c)
		}
		return
	}
	xInc := float64(dx) / float64(steps)
	yInc := float64(dy) / float64(steps)
	x, y := float64(x1), float64(y1)
	for i := 0; i <= steps; i++ {
		px, py := int(math.Round(x)), int(math.Round(y))
		if px >= 0 && px < r.width && py >= 0 && py < r.height {
			r.canvas.Set(px, py, c)
		}
		x += xInc
		y += yInc
	}
}

func (r *Renderer) fillPolygon(coords []datasource.Coordinate, fill color.Color) {
	if len(coords) < 3 || fill == nil {
		return
	}
	pixels := make([]struct{ x, y int }, len(coords))
	minPy, maxPy := r.height, 0
	for i, c := range coords {
		px, py := r.worldToPixel(c.X, c.Y)
		pixels[i] = struct{ x, y int }{px, py}
		if py < minPy {
			minPy = py
		}
		if py > maxPy {
			maxPy = py
		}
	}
	if minPy < 0 {
		minPy = 0
	}
	if maxPy >= r.height {
		maxPy = r.height - 1
	}

	for y := minPy; y <= maxPy; y++ {
		var xs []int
		for i := 0; i < len(pixels); i++ {
			j := (i + 1) % len(pixels)
			yi, yj := pixels[i].y, pixels[j].y
			if yi == yj {
				continue
			}
			if (yi <= y && y < yj) || (yj <= y && y < yi) {
				x := pixels[i].x + (y-yi)*(pixels[j].x-pixels[i].x)/(yj-yi)
				xs = append(xs, x)
			}
		}
		// Bubble sort intersections
		for i := 0; i < len(xs); i++ {
			for j := i + 1; j < len(xs); j++ {
				if xs[i] > xs[j] {
					xs[i], xs[j] = xs[j], xs[i]
				}
			}
		}
		for i := 0; i+1 < len(xs); i += 2 {
			for x := xs[i]; x <= xs[i+1] && x < r.width; x++ {
				if x >= 0 {
					r.canvas.Set(x, y, fill)
				}
			}
		}
	}
}

func (r *Renderer) Image(format string) ([]byte, error) {
	var buf bytes.Buffer
	switch format {
	case "image/jpeg":
		err := jpeg.Encode(&buf, r.canvas, &jpeg.Options{Quality: 90})
		return buf.Bytes(), err
	default:
		err := png.Encode(&buf, r.canvas)
		return buf.Bytes(), err
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
