package mqttm

import (
	"fmt"

	"go.uber.org/zap"
)

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
