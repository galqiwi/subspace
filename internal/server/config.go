package server

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Listen       string   `yaml:"listen"`
	ClientTokens []string `yaml:"client_tokens"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = file.Close()
	}()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	var output Config
	err = decoder.Decode(&output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %v: %w", path, err)
	}

	err = output.validate()
	if err != nil {
		return nil, fmt.Errorf("invalid config %v: %w", path, err)
	}
	return &output, nil
}

func (c *Config) validate() error {
	if c.Listen == "" {
		return errors.New("listen is not set")
	}
	if len(c.ClientTokens) == 0 {
		return errors.New("client_tokens is empty")
	}
	for i, token := range c.ClientTokens {
		if token == "" {
			return fmt.Errorf("client_tokens[%v] is empty", i)
		}
	}
	return nil
}
