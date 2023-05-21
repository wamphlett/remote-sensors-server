package config

// Manager defines configuration for the Manager
type Manager struct {
	Fields *ManagerFields `yaml:"fields"`
}

// ManagerFields defines the metrics and metadata fields which each device
// will record, any fields not listed in this config will be dropped by the manager
type ManagerFields struct {
	Metrics  []string `yaml:"metrics"`
	Metadata []string `yaml:"metadata"`
}
