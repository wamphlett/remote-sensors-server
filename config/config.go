package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Config defines all of the configuration to run the app
type Config struct {
	Server    *Server    `yaml:"server"`
	Manager   *Manager   `yaml:"manager"`
	Consumers *Consumers `yaml:"consumers"`
}

// NewConfigFromFile will create a new Config based on the given YAML file
func NewConfigFromFile(filepath string) *Config {
	if _, err := os.Stat(filepath); err != nil {
		fmt.Println("missing config file")
		os.Exit(1)
	}

	configFile, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Println("failed to read config file")
		os.Exit(1)
	}

	c := &Config{}
	err = yaml.Unmarshal(configFile, &c)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	return c
}

// Consumers holds all the configure for the consumers
type Consumers struct {
	MQTT *MQTT `yaml:"mqtt"`
}
