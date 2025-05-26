package mqttm

import (
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

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
