package wms

import (
	"encoding/xml"
	"math"

	"wms-server/internal/config"
	"wms-server/internal/datasource"
	"wms-server/internal/srs"
)

type WMS_Capabilities struct {
	XMLName    xml.Name   `xml:"WMS_Capabilities"`
	Version    string     `xml:"version,attr"`
	XMLNS      string     `xml:"xmlns,attr"`
	XMLNSXLink string     `xml:"xmlns:xlink,attr"`
	Service    Service    `xml:"Service"`
	Capability Capability `xml:"Capability"`
}

type Service struct {
	Name        string `xml:"Name"`
	Title       string `xml:"Title"`
	Abstract    string `xml:"Abstract"`
	OnlineResource struct {
		HREF string `xml:"xlink:href,attr"`
	} `xml:"OnlineResource"`
	MaxWidth  int `xml:"MaxWidth"`
	MaxHeight int `xml:"MaxHeight"`
}

type Capability struct {
	Request     Request     `xml:"Request"`
	Exception   Exception   `xml:"Exception"`
	Layer       Layer       `xml:"Layer"`
}

type Request struct {
	GetCapabilities Operation `xml:"GetCapabilities"`
	GetMap          Operation `xml:"GetMap"`
	GetFeatureInfo  Operation `xml:"GetFeatureInfo"`
}

type Operation struct {
	Format       []string     `xml:"Format"`
	DCPType      DCPType      `xml:"DCPType"`
}

type DCPType struct {
	HTTP HTTP `xml:"HTTP"`
}

type HTTP struct {
	Get  HTTPMethod  `xml:"Get"`
	Post HTTPMethod  `xml:"Post"`
}

type OnlineResource struct {
	HREF string `xml:"xlink:href,attr"`
}

type HTTPMethod struct {
	OnlineResource OnlineResource `xml:"OnlineResource"`
}

type Exception struct {
	Format []string `xml:"Format"`
}

type Layer struct {
	Name        string   `xml:"Name"`
	Title       string   `xml:"Title"`
	Abstract    string   `xml:"Abstract,omitempty"`
	CRS         []string `xml:"CRS"`
	BoundingBox []BBox   `xml:"BoundingBox,omitempty"`
	Layer       []Layer  `xml:"Layer,omitempty"`
}

type BBox struct {
	CRS  string  `xml:"CRS,attr"`
	MinX float64 `xml:"minx,attr"`
	MinY float64 `xml:"miny,attr"`
	MaxX float64 `xml:"maxx,attr"`
	MaxY float64 `xml:"maxy,attr"`
}

func GenerateCapabilities(cfg *config.Config, sources map[string]datasource.DataSource, baseURL string) ([]byte, error) {
	root := WMS_Capabilities{
		Version: ServiceVersion,
		XMLNS:      "http://www.opengis.net/wms",
		XMLNSXLink: "http://www.w3.org/1999/xlink",
		Service: Service{
			Name:     ServiceName,
			Title:    ServiceTitle,
			Abstract: "單機離線 WMS 1.3.0 發布伺服器",
			MaxWidth:  4096,
			MaxHeight: 4096,
		},
		Capability: Capability{
			Request: Request{
				GetCapabilities: Operation{
					Format: []string{"text/xml"},
					DCPType: DCPType{
						HTTP: HTTP{
							Get:  HTTPMethod{OnlineResource: OnlineResource{HREF: baseURL}},
							Post: HTTPMethod{OnlineResource: OnlineResource{HREF: baseURL}},
						},
					},
				},
				GetMap: Operation{
					Format: SupportedFormats,
					DCPType: DCPType{
						HTTP: HTTP{
							Get:  HTTPMethod{OnlineResource: OnlineResource{HREF: baseURL}},
							Post: HTTPMethod{OnlineResource: OnlineResource{HREF: baseURL}},
						},
					},
				},
				GetFeatureInfo: Operation{
					Format: []string{"text/plain", "text/xml"},
					DCPType: DCPType{
						HTTP: HTTP{
							Get:  HTTPMethod{OnlineResource: OnlineResource{HREF: baseURL}},
							Post: HTTPMethod{OnlineResource: OnlineResource{HREF: baseURL}},
						},
					},
				},
			},
			Exception: Exception{
				Format: []string{"text/plain", "text/xml"},
			},
			Layer: Layer{
				Name:  "root",
				Title: ServiceTitle,
				CRS:   SupportedCRS,
			},
		},
	}

	for _, lc := range cfg.Layers {
		subLayer := Layer{
			Name:     lc.Name,
			Title:    lc.Title,
			Abstract: lc.Abstract,
			CRS:      SupportedCRS,
		}
		// Try to get layer bbox from source metadata
		if src, ok := sources[lc.Name]; ok {
			if src.Type() == "mbtiles" || src.Type() == "xyz" {
				subLayer.BoundingBox = []BBox{
					{CRS: "EPSG:4326", MinX: -180, MinY: -90, MaxX: 180, MaxY: 90},
					{CRS: "EPSG:3857", MinX: -2 * math.Pi * srs.EarthRadius(), MinY: -2 * math.Pi * srs.EarthRadius(), MaxX: 2 * math.Pi * srs.EarthRadius(), MaxY: 2 * math.Pi * srs.EarthRadius()},
				}
			}
		}
		root.Capability.Layer.Layer = append(root.Capability.Layer.Layer, subLayer)
	}

	return xml.MarshalIndent(root, "", "  ")
}
