package wms

const (
	ServiceName    = "WMS"
	ServiceTitle   = "Offline WMS Server"
	ServiceVersion = "1.3.0"
	UpdateSequence = "0"

	ParamFormatPNG  = "image/png"
	ParamFormatJPEG = "image/jpeg"
)

var SupportedFormats = []string{ParamFormatPNG, ParamFormatJPEG}
var SupportedCRS = []string{"EPSG:4326", "EPSG:3857"}
