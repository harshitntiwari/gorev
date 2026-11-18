package config

import (
	"cmp"
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v4"
)

type LoadBalancingStrategy string

const (
	RoundRobin       LoadBalancingStrategy = "round_robin"
	LeastConnections LoadBalancingStrategy = "least_connections"
)

const (
	defaultListenPort            = 8080
	defaultLoadBalancingStrategy = RoundRobin
	defaultBackendWeight         = 1
)

type Config struct {
	Server    ServerConfig              `yaml:"server"`
	Upstreams map[string]UpstreamConfig `yaml:"upstreams"`
	Routes    []RouteConfig             `yaml:"routes"`
}

type ServerConfig struct {
	ListenPort int `yaml:"listen_port"`
}

type UpstreamConfig struct {
	// load balancing strategy
	Strategy    LoadBalancingStrategy `yaml:"strategy"`
	HealthCheck HealthCheck           `yaml:"health_check"`
	Backends    []BackendConfig       `yaml:"backends"`
}

type HealthCheck struct {
	Path     string        `yaml:"path"`
	Interval time.Duration `yaml:"interval"`
}

type BackendConfig struct {
	URL    string `yaml:"url"`
	Weight int    `yaml:"weight"`
}

type RouteConfig struct {
	Host       string `yaml:"host"`
	PathPrefix string `yaml:"path_prefix"`
	Upstream   string `yaml:"upstream"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			ListenPort: defaultListenPort,
		},
		Upstreams: make(map[string]UpstreamConfig),
	}
}

// Load reads the user defined YAML file and overrides the default config
func Load(filePath string) (*Config, error) {
	cfg := DefaultConfig()

	f, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if err := yaml.Load(f, cfg); err != nil {
		return nil, err
	}

	if err := validateCfg(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func validateCfg(cfg *Config) error {
	// validate routes
	for _, route := range cfg.Routes {
		if _, exists := cfg.Upstreams[route.Upstream]; !exists {
			return fmt.Errorf("routes reference an upstream that does not exist in upstreams: %s", route.Upstream)
		}
	}

	// validate upstreams
	for name, upstream := range cfg.Upstreams {
		if len(upstream.Backends) == 0 {
			return fmt.Errorf("upstream %s has no backend configured", name)
		}
		for _, backend := range upstream.Backends {
			backend.Weight = cmp.Or(backend.Weight, int(defaultBackendWeight))
		}

		upstream.Strategy = cmp.Or(upstream.Strategy, defaultLoadBalancingStrategy)
		upstream.HealthCheck.Interval = cmp.Or(upstream.HealthCheck.Interval, 10*time.Second)
		cfg.Upstreams[name] = upstream
	}
	return nil
}
