package mqttm

import (
	"context"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

var (
	connectStatus string = "off-line"
)

const (
	RECONNECT_INTERVAL_SEC time.Duration = 5000 * time.Millisecond
	KEEP_ALIVE_SEC         time.Duration = 60000 * time.Millisecond

	DEFAULT_QOS byte = 0

	QUEUE_SIZE            int = 16
	REGISTER_TOPIC_PREFIX     = "register"
)

type (
	// Config holds the MQTT configuration
	Config struct {
		Endpoint        string          `json:"endpoint"`
		Username        string          `json:"username"`
		Password        string          `json:"password"`
		RootCA          string          `json:"root_ca"`
		PrivateKey      string          `json:"key"`
		ClientCert      string          `json:"cert"`
		SubscribeTopics map[string]byte `json:"subscribe_topics"`
	}

	// Module represents the MQTT module with its configuration and handlers
	Module struct {
		ctx      context.Context
		clientID string
		hostName string
		client   MQTT.Client
		option   MQTT.ClientOptions

		PubCh chan Contents
		SubCh chan Contents
	}

	// Contents represents the message contents for publishing and subscribing
	Contents struct {
		Timestamp time.Time `json:"timestamp"`
		Hostname  string    `json:"hostname"`
		Topic     string    `json:"topic"`
		ClientID  string    `json:"client_id"`
		QoS       byte      `json:"qos"`
		Payload   []byte    `json:"payload"`
	}

	// Handler defines the interface for handling MQTT messages
	Handler interface {
		New(ctx context.Context, clientID string, conf Config) (*Module, error)
		Run() error
		Stop()
	}
)
