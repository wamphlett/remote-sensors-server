package mqtt

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tidwall/gjson"

	"github.com/wamphlett/remote-sensors-server/config"
	"github.com/wamphlett/remote-sensors-server/pkg/manager"
)

type topicDetails struct {
	tester   *regexp.Regexp
	mappings *config.MQTTMapping
}

// Consumer defines a new MQTT consumer
type Consumer struct {
	client  mqtt.Client
	manager *manager.Manager

	topics map[string]*topicDetails
}

// New creates a fully configured Consumer
func New(cfg *config.MQTT, manager *manager.Manager) *Consumer {
	c := &Consumer{
		manager: manager,
		topics:  make(map[string]*topicDetails),
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.Host, cfg.Port))
	opts.SetClientID("remote-sensors-server")
	opts.SetDefaultPublishHandler(c.consumeFunc)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	c.client = client

	c.configureTopics(cfg.Topics)

	return c
}

func (c *Consumer) configureTopics(topics config.MQTTTopics) {
	i := 0
	for topic, mapping := range topics {
		// validate the fopic format
		if strings.Contains(topic, "#") {
			fmt.Println("topic wild cards are not supported")
			os.Exit(1)
		}

		if !strings.Contains(topic, "{DEVICE}") {
			fmt.Println("Missing {DEVICE} from tpoic")
			os.Exit(1)
		}
		// replace the device with a wildcard
		formattedTopic := strings.Replace(topic, "{DEVICE}", "#", 1)

		// create the regex which will be used to identify the incoming devide
		parts := strings.Split(formattedTopic, "#")
		regexString := regexp.QuoteMeta(parts[0])
		regexString += `([\w\d\-]+)`
		if len(parts) > 1 {
			regexString += regexp.QuoteMeta(parts[1])
		}
		r, err := regexp.Compile(regexString)
		if err != nil {
			fmt.Println("failed to generate topic regular expression")
			os.Exit(1)
		}
		c.topics[topic] = &topicDetails{
			tester:   r,
			mappings: mapping,
		}

		// subscribe to the topic, MQTT topics must not have anything after the wildcard
		topicToSubscribe := parts[0] + "#"
		token := c.client.Subscribe(topicToSubscribe, 1, nil)
		token.Wait()
		log.Printf("Subscribed to topic %s\n", topicToSubscribe)
		i++
	}
}

// consumeFunc is responsible for reading a message from the broker and mapping the incoming
// fields to the correct device
func (c *Consumer) consumeFunc(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())

	matches := []string{}
	for _, topic := range c.topics {
		matches = topic.tester.FindStringSubmatch(msg.Topic())
		if len(matches) == 2 {
			break
		}
	}

	if len(matches) != 2 {
		return
	}

	matchedTopic := strings.Replace(msg.Topic(), matches[1], "{DEVICE}", 1)

	updates := manager.Updates{
		Metrics:  map[string]float64{},
		Metadata: map[string]string{},
	}
	for k, m := range c.topics[matchedTopic].mappings.Metrics {
		r := gjson.Get(string(msg.Payload()), k)
		if r.Exists() {
			updates.Metrics[m] = r.Float()
		}
	}

	for k, m := range c.topics[matchedTopic].mappings.Metadata {
		r := gjson.Get(string(msg.Payload()), k)
		if r.Exists() {
			updates.Metadata[m] = r.String()
		}
	}

	if len(updates.Metrics) > 0 || len(updates.Metadata) > 0 {
		c.manager.Update(matches[1], updates)
	}
}
