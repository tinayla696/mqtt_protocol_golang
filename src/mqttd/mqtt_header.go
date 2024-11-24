package mqttd

import (
	"context"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

const (
	defaultRetryCount    int           = 5
	defaultWaitTime      time.Duration = 250 * time.Second
	defKeepAliveTime     time.Duration = 300 * time.Second
	defReconnectInterval time.Duration = 500 * time.Millisecond

	statusTopic string = "status"
)

var (
	clientStatus string = "power-off"
)

/*-----------------------------------*/ /*
	Class
*/ /*-----------------------------------*/
type MqttClient struct {
	ctx    context.Context
	option *MQTT.ClientOptions
	client MQTT.Client

	subTopics []string

	// Ch
	PublishCh   chan Contents
	SubscribeCh chan Contents
}

/*-----------------------------------*/ /*
	Options
*/ /*-----------------------------------*/
type MqttConf struct {
	Endpoint              string   `json:"endpoint"`
	RootCA                string   `json:"rootca"`
	PrivateKey            string   `json:"key"`
	ClientCert            string   `json:"cert"`
	SubscribeTopicPrefixs []string `json:"subscribetopicprefixs"`
}

/*-----------------------------------*/ /*
	MQTT Data-Contents pack
*/ /*-----------------------------------*/
type Contents struct {
	TimeStamp   time.Time
	TopicPrefix string
	Payload     map[string]interface{}
}
