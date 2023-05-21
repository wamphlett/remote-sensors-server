package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/wamphlett/remote-sensors-server/config"
	"github.com/wamphlett/remote-sensors-server/pkg/manager"
	"github.com/wamphlett/remote-sensors-server/pkg/mqtt"
	"github.com/wamphlett/remote-sensors-server/pkg/server"
)

func main() {
	// load the config from file
	cfg := config.NewConfigFromFile("./config.yaml")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// create a new device manager
	deviceManager := manager.New(cfg.Manager)

	// start any mqtt consumers
	if cfg.Consumers.MQTT != nil {
		mqtt.New(cfg.Consumers.MQTT, deviceManager)
	}

	// start the server
	server := server.New(cfg.Server, deviceManager)
	server.Start()

	<-sigChan
	server.Shutdown()
	deviceManager.Shutdown()
}
