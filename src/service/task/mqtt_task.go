package task

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tinayla696/mqtt_protocol_golang/module/mqttm"
	"go.uber.org/zap"
)

// *--------------------------------------------------------------------------------------
// MqttTask
type MqttTask struct {
	Contents mqttm.Contents
	ID       int
}

// *--------------------------------------------------------------------------------------
// Execute
func (t *MqttTask) Execute(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Execute the MQTT task
		zap.S().Debugf("Executing MqttTask ID: %d, Topic: %s", t.ID, t.Contents.Topic)

		var jsonPayload map[string]interface{}
		if err := json.Unmarshal(t.Contents.Payload, &jsonPayload); err != nil {
			zap.S().Errorf("Failed to unmarshal payload for MqttTask ID: %d, Error: %v", t.ID, err)
			zap.S().Infof("Payload: %s", string(t.Contents.Payload))
			return fmt.Errorf("failed to unmarshal payload: %w", err)
		}
		zap.S().Infof("Payload: %+v", jsonPayload)
		return nil
	}
}

// * --------------------------------------------------------------------------------------
// String
func (t *MqttTask) String() string {
	return fmt.Sprintf("MqttTask{Topic: %s, ID: %d}", t.Contents.Topic, t.ID)
}

// *--------------------------------------------------------------------------------------
// Type
func (t *MqttTask) Type() TaskType {
	return MqttTaskType
}
