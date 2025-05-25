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
func (m *Module) getStatusPayload() []byte {
	statusMsg := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"client_id": m.clientID,
		"Status":    connectStatus,
	}
	payload, _ := json.Marshal(statusMsg)
	return payload
}

// *--------------------------------------------------------------------------------------
func (m *Module) connectToBroker() error {
	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	zap.S().Infof("Connected to MQTT broker: %s", m.clientID)
	return nil
}

// *--------------------------------------------------------------------------------------
func (m *Module) connectHandler(subTopics map[string]byte) MQTT.OnConnectHandler {
	return func(client MQTT.Client) {
		zap.S().Infof("Connecting to MQTT broker: %+v", m.option.Servers)
		connectStatus = "on-line"
		topic := fmt.Sprintf("%s/%s", REGISTER_TOPIC_PREFIX, m.clientID)
		if err := m.publishFn(topic, byte(0), m.getStatusPayload()); err != nil {
			zap.S().Errorf("Failed to publish status message: %v", err)
		}

		// Set up subscriptions
		if token := client.SubscribeMultiple(subTopics, m.subscribeFn); token.Wait() && token.Error() != nil {
			zap.S().Errorf("Failed to subscribe to topics: %v", token.Error())
		} else {
			zap.S().Infof("Subscribed to topics: %+v", subTopics)
		}
	}
}

// *--------------------------------------------------------------------------------------
func (m *Module) disconnectFromBroker() {
	connectStatus = "off-line"
	topic := fmt.Sprintf("%s/%s", REGISTER_TOPIC_PREFIX, m.clientID)
	if err := m.publishFn(topic, byte(0), m.getStatusPayload()); err != nil {
		zap.S().Errorf("Failed to publish disconnection message: %v", err)
	}

	m.client.Disconnect(250)
	zap.S().Infof("Disconnecting from MQTT broker: %s", m.clientID)
}

// *--------------------------------------------------------------------------------------
func (m *Module) publishFn(topic string, qos byte, payload []byte) error {
	if qos > 2 {
		zap.S().Warnf("QoS level %d is not supported, using QoS 0", qos)
		qos = DEFAULT_QOS
	}
	token := m.client.Publish(topic, qos, false, payload)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to publish message to topic %s: %w", topic, token.Error())
	}
	zap.S().Debugf("Published message to topic %s with QoS %d", topic, qos)
	return nil
}

// *--------------------------------------------------------------------------------------
func (m *Module) subscribeFn(client MQTT.Client, contents MQTT.Message) {
	m.SubCh <- Contents{
		Timestamp: time.Now(),
		Hostname:  m.hostName,
		Topic:     contents.Topic(),
		ClientID:  m.clientID,
		QoS:       contents.Qos(),
		Payload:   contents.Payload(),
	}
}

// *--------------------------------------------------------------------------------------
func (m *Module) publishLoop() {
	for {
		select {
		case <-m.PubCh:
			close(m.PubCh)
			return

		case contents, isNotClose := <-m.PubCh:
			if !isNotClose {
				return
			}
			topic := fmt.Sprintf("%s/%s", contents.Topic, m.clientID)
			if err := m.publishFn(topic, contents.QoS, contents.Payload); err != nil {
				zap.S().Errorf("Failed to publish message: %v", err)
			}
		}
	}
}
