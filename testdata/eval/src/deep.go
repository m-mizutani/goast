package sample

import "fmt"

// Config exercises composite literals, struct tags and nested expressions.
type Config struct {
	Name    string            `json:"name"`
	Retries int               `json:"retries"`
	Labels  map[string]string `json:"labels"`
}

type Runner interface {
	Start(cfg *Config) error
}

func buildConfig(depth int) *Config {
	cfg := &Config{
		Name:    "corpus",
		Retries: 3,
		Labels: map[string]string{
			"tier": "test",
			"lang": "go",
		},
	}

	if depth > 0 {
		if depth > 1 {
			if depth > 2 {
				fmt.Println("deeply nested call", cfg.Name)
			}
		}
	}

	values := []int{1, 2, 3}
	sum := values[0] + values[1]*values[2] - len(cfg.Name)
	cfg.Retries = sum

	return cfg
}
