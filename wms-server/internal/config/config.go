package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type StyleConfig struct {
	FillColor   string  `yaml:"fill_color"`
	StrokeColor string  `yaml:"stroke_color"`
	LineWidth   float64 `yaml:"line_width"`
	PointSize   int     `yaml:"point_size"`
}

type LayerConfig struct {
	Name     string       `yaml:"name"`
	Title    string       `yaml:"title"`
	Abstract string       `yaml:"abstract"`
	Type     string       `yaml:"type"`
	Path     string       `yaml:"path"`
	CRS      string       `yaml:"crs"`
	Style    *StyleConfig `yaml:"style,omitempty"`
}

type Config struct {
	Server ServerConfig  `yaml:"server"`
	Layers []LayerConfig `yaml:"layers"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":8080"
	}
	for i := range cfg.Layers {
		if cfg.Layers[i].CRS == "" {
			cfg.Layers[i].CRS = "EPSG:4326"
		}
	}
	return &cfg, nil
}
