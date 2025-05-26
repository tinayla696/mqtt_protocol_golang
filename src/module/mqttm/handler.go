package mqttm

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

// *--------------------------------------------------------------------------------------
// New
func New(ctx context.Context, clientID, hostname string, conf Config) (*Module, error) {
	module := &Module{
		ctx:      ctx,
		clientID: clientID,
		hostName: hostname,
		PubCh:    make(chan Contents, QUEUE_SIZE),
		SubCh:    make(chan Contents, QUEUE_SIZE),
	}
	option, err := module.setOptions(conf)
	if err != nil {
		return module, err
	}
	option.SetOnConnectHandler(module.connectHandler(conf.SubscribeTopics))
	option.SetConnectionLostHandler(func(client MQTT.Client, err error) {
		zap.S().Errorf("Connection lost: %v", err)
	})
	module.option = *option
	module.client = MQTT.NewClient(option)

	return module, nil
}

// *--------------------------------------------------------------------------------------
// Run
func (m *Module) Run() error {
	if m.client == nil || len(m.option.Servers) < 1 {
		return fmt.Errorf("MQTT client is already running or no servers configured")
	}

	if err := m.connectToBroker(); err != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", err)
	}

	go m.publishLoop()
	return nil
}

// *--------------------------------------------------------------------------------------
// Stop
func (m *Module) Stop() {
	m.disconnectFromBroker()
}

// *--------------------------------------------------------------------------------------
// Setupt MQTT options
func (m *Module) setOptions(conf Config) (*MQTT.ClientOptions, error) {
	if conf.Endpoint == "" {
		return &MQTT.ClientOptions{}, fmt.Errorf("MQTT endpoint is required")
	}

	if m.clientID == "" {
		return &MQTT.ClientOptions{}, fmt.Errorf("MQTT client ID is required")
	}

	// option
	option := MQTT.NewClientOptions()
	option.SetClientID(m.clientID)
	if conf.Username != "" && conf.Password != "" {
		option.SetUsername(conf.Username)
		option.SetPassword(conf.Password)
	}
	option.SetKeepAlive(KEEP_ALIVE_SEC)
	option.SetMaxReconnectInterval(RECONNECT_INTERVAL_SEC)
	option.SetAutoReconnect(true)
	option.SetCleanSession(true)

	if conf.RootCA != "" && conf.PrivateKey != "" && conf.ClientCert != "" {
		certPool := x509.NewCertPool()
		prmCerts, err := os.ReadFile(conf.RootCA)
		if err != nil {
			return option, err
		}

		certPool.AppendCertsFromPEM(prmCerts)
		cer, err := tls.LoadX509KeyPair(conf.ClientCert, conf.PrivateKey)
		if err != nil {
			return option, err
		}
		option.SetTLSConfig(&tls.Config{
			RootCAs:            certPool,
			Certificates:       []tls.Certificate{cer},
			InsecureSkipVerify: true, // For testing purposes, set to false in production
			MinVersion:         tls.VersionTLS12,
		})
		option.AddBroker("ssl://" + conf.Endpoint)
		return option, err
	}

	// TLS
	option.AddBroker("tcp://" + conf.Endpoint)
	return option, nil
}

// *--------------------------------------------------------------------------------------
// Get Connect Status Payload
func (m *Module) getStatusPayload() []byte {
	statusMsg := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"id":        m.clientID,
		"Status":    connectStatus,
	}
	payload, _ := json.Marshal(statusMsg)
	return payload
}
