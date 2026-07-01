package server

import (
	"context"
	"encoding/xml"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"wms-server/internal/arcgis"
	"wms-server/internal/config"
	"wms-server/internal/datasource"
	"wms-server/internal/wms"
)

type Server struct {
	cfg     *config.Config
	sources map[string]datasource.DataSource
	http    *http.Server
}

func New(cfg *config.Config) (*Server, error) {
	sources := make(map[string]datasource.DataSource)
	for _, layer := range cfg.Layers {
		src, err := datasource.FromConfig(layer)
		if err != nil {
			return nil, fmt.Errorf("初始化圖資層 %s 失敗: %w", layer.Name, err)
		}
		sources[layer.Name] = src
		log.Printf("已載入圖資層: %s (類型: %s)", layer.Name, layer.Type)
	}

	s := &Server{
		cfg:     cfg,
		sources: sources,
	}

	arcHandler := arcgis.NewHandler(cfg, sources)
	mux := http.NewServeMux()
	mux.Handle("/arcgis/", arcHandler)
	mux.HandleFunc("/wms", s.handleWMS)
	mux.HandleFunc("/", s.handleRoot)

	s.http = &http.Server{
		Addr:         cfg.Server.Listen,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	return s, nil
}

func (s *Server) Start() error {
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s.http.Shutdown(ctx)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s %v", r.Method, r.URL.Path, r.URL.RawQuery, time.Since(start))
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>WMS Server</title></head>
<body>
<h1>WMS 1.3.0 Server</h1>
<p>已離線執行中。</p>
<ul>
	<li><a href="/wms?SERVICE=WMS&amp;VERSION=1.3.0&amp;REQUEST=GetCapabilities">GetCapabilities</a></li>
</ul>
<h2>已載入圖資層</h2>
<ul>`)
	for _, l := range s.cfg.Layers {
		fmt.Fprintf(w, "<li>%s (%s) - %s</li>", l.Name, l.Type, l.Title)
	}
	fmt.Fprint(w, `</ul></body></html>`)
}

func (s *Server) handleWMS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "僅支援 GET 請求", http.StatusMethodNotAllowed)
		return
	}

	params := r.URL.Query()
	service := strings.ToUpper(params.Get("SERVICE"))
	request := strings.ToUpper(params.Get("REQUEST"))

	if service != "WMS" {
		s.writeError(w, "缺少或無效的 SERVICE 參數", "text/plain")
		return
	}

	switch request {
	case "GETCAPABILITIES":
		s.handleCapabilities(w, r)
	case "GETMAP":
		s.handleMap(w, params)
	case "GETFEATUREINFO":
		s.handleFeatureInfo(w, params)
	default:
		s.writeError(w, fmt.Sprintf("不支援的 REQUEST: %s", request), "text/plain")
	}
}

func (s *Server) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)

	xmlData, err := wms.GenerateCapabilities(s.cfg, s.sources, baseURL)
	if err != nil {
		s.writeError(w, fmt.Sprintf("產生 Capabilities 失敗: %v", err), "text/plain")
		return
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Write([]byte(xml.Header))
	w.Write(xmlData)
}

func (s *Server) handleMap(w http.ResponseWriter, params url.Values) {
	data, format, err := wms.HandleGetMap(params, s.sources, layerNames(s.cfg.Layers))
	if err != nil {
		s.writeError(w, err.Error(), "text/plain")
		return
	}
	w.Header().Set("Content-Type", format)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Write(data)
}

func (s *Server) handleFeatureInfo(w http.ResponseWriter, params url.Values) {
	result, err := wms.HandleGetFeatureInfo(params, s.sources)
	if err != nil {
		s.writeError(w, err.Error(), "text/plain")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, result)
}

func (s *Server) writeError(w http.ResponseWriter, msg string, format string) {
	log.Printf("WMS 錯誤: %s", msg)
	w.Header().Set("Content-Type", format)
	http.Error(w, msg, http.StatusBadRequest)
}

func layerNames(layers []config.LayerConfig) []string {
	names := make([]string, len(layers))
	for i, l := range layers {
		names[i] = l.Name
	}
	return names
}
