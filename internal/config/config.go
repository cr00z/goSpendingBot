package config

import (
	"os"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const configFile = "build/bot/config.yaml"

type Config struct {
	Token             string `yaml:"token"`
	CurrencyCacheSize int    `yaml:"currency_cache_size"`
	ReportCacheSize   int    `yaml:"report_cache_size"`
}

type Service struct {
	config Config
}

func New() (*Service, error) {
	s := &Service{}

	rawYAML, err := os.ReadFile(configFile)
	if err != nil {
		return nil, errors.Wrap(err, "reading config file")
	}

	err = yaml.Unmarshal(rawYAML, &s.config)
	if err != nil {
		return nil, errors.Wrap(err, "parsing yaml")
	}

	return s, nil
}

func (s *Service) Token() string {
	return s.config.Token
}

func (s *Service) CurrencyCacheSize() int {
	return s.config.CurrencyCacheSize
}

func (s *Service) ReportCacheSize() int {
	return s.config.ReportCacheSize
}
