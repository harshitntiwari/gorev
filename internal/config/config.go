package config

import (
	"cmp"
	"fmt"
	"net/url"
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

	// validate server config
	if cfg.Server.ListenPort < 1 || cfg.Server.ListenPort > 65535 {
		return fmt.Errorf("server listen port out of range, valid range: 1 - 65535")
	}

	// validate upstreams
	for name, upstream := range cfg.Upstreams {
		if len(upstream.Backends) == 0 {
			return fmt.Errorf("upstream %s has no backend configured", name)
		}

		for i, backend := range upstream.Backends {
			if _, err := url.Parse(backend.URL); err != nil {
				return fmt.Errorf("invalid backend url: %s", backend.URL)
			}
			if backend.Weight < 0 {
				return fmt.Errorf("invalid weight %d, backend weight must be a positive non-zero value", backend.Weight)
			}
			backend.Weight = cmp.Or(backend.Weight, int(defaultBackendWeight))
			upstream.Backends[i] = backend
		}

		upstream.Strategy = cmp.Or(upstream.Strategy, defaultLoadBalancingStrategy)
		if upstream.HealthCheck.Path != "" {
			upstream.HealthCheck.Interval = cmp.Or(upstream.HealthCheck.Interval, 10*time.Second)
		}
		cfg.Upstreams[name] = upstream
	}

	// validate routes
	for _, route := range cfg.Routes {
		if route.PathPrefix == "" {
			return fmt.Errorf("route path_prefix cannot be empty")
		}
		if _, exists := cfg.Upstreams[route.Upstream]; !exists {
			return fmt.Errorf("routes reference an upstream that does not exist in upstreams: %s", route.Upstream)
		}
	}
	return nil
}
