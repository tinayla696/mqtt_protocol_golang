package mqttm

import (
	"fmt"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

// *--------------------------------------------------------------------------------------
func (m *Module) connectToBroker() error {
	if token := m.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	zap.S().Infof("Connected to MQTT broker: %s", m.hostName)
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
