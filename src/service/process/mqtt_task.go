package process

import (
	"encoding/json"
	"fmt"

	"github.com/tinayla696/mqtt_protocol_golang/module/mqttm"
)

type MqttProcessingTask struct {
	Contents mqttm.Contents
}

func (c *MqttProcessingTask) Process() {
	fmt.Printf("Processing MQTT task with contents: %+v\n", c.Contents)

	var data map[string]interface{}
	if err := json.Unmarshal(c.Contents.Payload, &data); err != nil {
		fmt.Printf("Error unmarshalling MQTT contents: %v\n", err)
		return
	}

}
