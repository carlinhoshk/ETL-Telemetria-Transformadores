package messaging

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Publisher is a thin QoS-1 MQTT publisher used by the simulator.
type Publisher struct {
	client mqtt.Client
}

// NewPublisher connects to the broker. broker is a URL such as
// tcp://localhost:1883.
func NewPublisher(broker, clientID string) (*Publisher, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetCleanSession(true).
		SetConnectTimeout(5 * time.Second).
		SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("connect to %s: %w", broker, token.Error())
	}
	return &Publisher{client: client}, nil
}

// Publish sends payload on topic with QoS 1 (at-least-once).
func (p *Publisher) Publish(topic string, payload []byte) error {
	token := p.client.Publish(topic, 1, false, payload)
	token.Wait()
	return token.Error()
}

// Close disconnects, flushing pending messages.
func (p *Publisher) Close() {
	if p.client != nil && p.client.IsConnected() {
		p.client.Disconnect(250)
	}
}