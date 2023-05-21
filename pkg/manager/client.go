package manager

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/wamphlett/remote-sensors-server/config"
	broadcastServer "github.com/wamphlett/remote-sensors-server/pkg/broadcaster"
)

// Metric defines the methods required by metrics
type Metric interface {
	Update(float64)
}

// Updates is used to record incoming updates for a device
type Updates struct {
	Metrics  map[string]float64
	Metadata map[string]string
}

// Device holds information about a specific device
type Device struct {
	lastUpdate  time.Time
	broadcaster Broadcaster
	metrics     map[string]Metric
	metadata    map[string]string
}

// Broadcaster defines the methods required from a broadcast server
type Broadcaster interface {
	Publish([]byte)
	Subscribe() <-chan []byte
	Unsubscribe(<-chan []byte)
	ActiveSubscribers() int
	Shutdown()
}

// Manager stores information about devices in memory, it provides a way to update devices
// and lets clients subscribe to those updates
type Manager struct {
	sync.Mutex
	devices map[string]*Device
	fields  *config.ManagerFields
	stop    chan bool
}

// New creates a new fully configured Manager
func New(cfg *config.Manager) *Manager {
	m := &Manager{
		devices: make(map[string]*Device),
		stop:    make(chan bool),
		fields:  cfg.Fields,
	}

	go m.clean()

	return m
}

func (m *Manager) clean() {
	ticker := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			m.removeStaleDevices()
		}
	}
}

// removeStaleDevices will remove any device from the manager when it has not received
// any recent updates and does not have any current subscribers
func (m *Manager) removeStaleDevices() {
	m.Lock()
	defer m.Unlock()
	for deviceID, device := range m.devices {
		if time.Since(device.lastUpdate) > time.Minute && device.broadcaster.ActiveSubscribers() == 0 {
			log.Printf("removing stale device: %s\n", deviceID)
			delete(m.devices, deviceID)
		}
	}
}

// Update will apply the incoming updates to the given device ID
func (m *Manager) Update(deviceID string, updates Updates) error {
	device := m.getOrCreateDevice(deviceID)
	device.lastUpdate = time.Now()

	// pluck all of the updates
	for metricKey, newValue := range updates.Metrics {
		if metric, ok := device.metrics[metricKey]; ok {
			metric.Update(newValue)
		}
	}

	for metadataKey, newValue := range updates.Metadata {
		device.metadata[metadataKey] = newValue
	}

	m.publish(device)
	return nil
}

type payload struct {
	Metadata map[string]string `json:"metadata"`
	Metrics  map[string]Metric `json:"metrics"`
}

func (m *Manager) publish(d *Device) {
	startTime := time.Now()

	b, err := json.Marshal(payload{
		Metadata: d.metadata,
		Metrics:  d.metrics,
	})
	if err != nil {
		log.Println("Failed to marshal payload")
		return
	}

	marshalTime := time.Since(startTime)
	log.Printf("marshaled in: %d\n", marshalTime.Microseconds())

	d.broadcaster.Publish(b)
}

// Subscribe will return a channel the client can use to received updates about
// the given device
func (m *Manager) Subscribe(deviceID string) <-chan []byte {
	device := m.getOrCreateDevice(deviceID)
	return device.broadcaster.Subscribe()
}

// Unsubscribe will remove the given channel from the device to to prevent the
// client receiving any further updates
func (m *Manager) Unsubscribe(deviceID string, c <-chan []byte) {
	device := m.getOrCreateDevice(deviceID)
	device.broadcaster.Unsubscribe(c)
}

// Shutdown will stop the manager and close all subscriptions
func (m *Manager) Shutdown() {
	m.stop <- true
	for _, device := range m.devices {
		device.broadcaster.Shutdown()
	}
}

// getOrCreateDevice will ensure the given device is present on the manager and return it
func (m *Manager) getOrCreateDevice(deviceID string) *Device {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.devices[deviceID]; !ok {
		metrics := make(map[string]Metric)
		for _, k := range m.fields.Metrics {
			metrics[k] = NewAggregatedMetric()
		}

		m.devices[deviceID] = &Device{
			broadcaster: broadcastServer.New[[]byte](),
			metrics:     metrics,
			metadata:    make(map[string]string),
		}

		log.Printf("registered new device: %s\n", deviceID)
	}

	return m.devices[deviceID]
}
