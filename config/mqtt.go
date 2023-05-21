package config

// MQTTTopics defines a list of topics to subscribe to
type MQTTTopics map[string]*MQTTMapping

// MQTT contains the details to subscrible to an MQTT broker
type MQTT struct {
	Host   string     `yaml:"brokerHost"`
	Port   int        `yaml:"brokerPort"`
	Topics MQTTTopics `yaml:"topics"`
}

// MQTTMapping defines a list of mappings used to map incoming fields to
// manager fields
type MQTTMapping struct {
	Metrics  map[string]string `yaml:"metrics"`
	Metadata map[string]string `yaml:"metadata"`
}
