package mqttd

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"
)

/*--------------------------------------------------------------------------------------------------*/ /*
	Constructor
*/ /*--------------------------------------------------------------------------------------------------*/
func SetMqttClient(ctx context.Context, clientID string, conf MqttConf) (*MqttClient, error) {
	mqttOptions, err := setMqttOption(clientID, conf)
	if err != nil {
		return &MqttClient{}, err
	}

	return &MqttClient{
		ctx:         ctx,
		option:      mqttOptions,
		client:      MQTT.NewClient(mqttOptions),
		PublishCh:   make(chan Contents),
		SubscribeCh: make(chan Contents, 8),
		subTopics:   conf.SubscribeTopicPrefixs,
	}, nil
}

/*---------------------------------------------------------------------------------------------*/ /*
	Connection Option
*/ /*---------------------------------------------------------------------------------------------*/
func setMqttOption(clientID string, conf MqttConf) (*MQTT.ClientOptions, error) {
	option := MQTT.NewClientOptions()
	if clientID == "" {
		return option, errors.New("client id not specified")
	}
	option.SetClientID(clientID)
	option.SetKeepAlive(defKeepAliveTime)
	option.SetMaxReconnectInterval(defReconnectInterval)
	option.SetAutoReconnect(true)

	// SSL
	if conf.RootCA != "" && conf.ClientCert != "" && conf.PrivateKey != "" {
		// Set Endpoint
		option.AddBroker(fmt.Sprintf("ssl://%s", conf.Endpoint))

		// Check RootCA
		certPool := x509.NewCertPool()
		pemCerts, err := os.ReadFile(conf.RootCA)
		if err != nil {
			return option, err
		}

		// Check Cert and Key
		certPool.AppendCertsFromPEM(pemCerts)
		cer, err := tls.LoadX509KeyPair(conf.ClientCert, conf.PrivateKey)
		if err != nil {
			return option, err
		}

		// Set SSL option
		option.SetTLSConfig(&tls.Config{
			RootCAs:      certPool,
			Certificates: []tls.Certificate{cer},
		})
		zap.S().Info(fmt.Sprintf("MQTT Broker Addr: ssl://%s\n", conf.Endpoint))
		return option, nil
	}

	// TLS
	option.AddBroker(fmt.Sprintf("tcp://%s", conf.Endpoint))
	zap.S().Info(fmt.Printf("MQTT Broker Addr: tcp://%s", conf.Endpoint))
	return option, nil
}

/*------------------------------------------------------------------------------*/ /*
	Start MQTT Pub/Sub Process
*/ /*------------------------------------------------------------------------------*/
func (mqtt *MqttClient) Start() {
	go mqtt.connectToMQTT()
}

/*------------------------------------------------------------------------------*/ /*
	Stop MQTT Pub/Sub Process
*/ /*------------------------------------------------------------------------------*/
func (mqtt *MqttClient) End() {
	mqtt.disconnectToMQTT()
}

/*------------------------------------------------------------------------------*/ /*
	Get client status return `payload`
*/ /*------------------------------------------------------------------------------*/
func (mqtt *MqttClient) getStatus() map[string]interface{} {
	return map[string]interface{}{
		"TimeStamp": time.Now().Format(time.RFC3339),
		"ID":        mqtt.option.ClientID,
		"Status":    clientStatus,
	}
}

/*------------------------------------------------------------------------------*/ /*
	MQTT Connection
*/ /*------------------------------------------------------------------------------*/
func (mqtt *MqttClient) connectToMQTT() {
	// Broker connection
	for {
		if token := mqtt.client.Connect(); token.Wait() && token.Error() != nil {
			zap.S().Warn("Failed to connect MQTT Broker. [", mqtt.option.Servers, "]")
			zap.S().Info("Try reconnection.......")
			continue
		}
		break
	}

	// Connect successfull
	zap.S().Info("MQTT Broker successfull. [", mqtt.option.Servers, "]")
	clientStatus = "on-line"
	topic := fmt.Sprintf("%s/%s", mqtt.option.ClientID, statusTopic)
	payload, _ := json.Marshal(mqtt.getStatus())
	mqtt.publishFn(topic, payload)

	// Start function loop
	go mqtt.publishLoop()
	if len(mqtt.subTopics) == 1 {
		go mqtt.subscribeLoop(mqtt.subTopics[0])
	} else {
		go mqtt.multipleSubscribeLoop(mqtt.subTopics)
	}
}

/*------------------------------------------------------------------------------*/ /*
	MQTT Disconnection
*/ /*------------------------------------------------------------------------------*/
func (mqtt *MqttClient) disconnectToMQTT() {
	clientStatus = "off-line"
	topic := fmt.Sprintf("%s/%s", mqtt.option.ClientID, statusTopic)
	payload, _ := json.Marshal(mqtt.getStatus())
	mqtt.publishFn(topic, payload)

	// Close connection
	if mqtt.client.IsConnected() {
		mqtt.client.Disconnect(500)
	}
	zap.S().Info("Close connection. [", mqtt.option.Servers, "]")

	// Close Ch
	close(mqtt.SubscribeCh)
}

/*------------------------------------------------------------------------------*/ /*
	Retry handler
*/ /*------------------------------------------------------------------------------*/
func retryhandler(retryCount int, fn func() error) error {
	var err error
	for retryCount > 0 {
		retryCount--
		if err = fn(); err == nil {
			return nil
		}
		<-time.After(time.Duration(defaultRetryCount) * time.Second)
	}
	return err
}

/*------------------------------------------------------------------------------*/ /*
	Publish handler
*/ /*------------------------------------------------------------------------------*/
func (mqtt *MqttClient) publishFn(topic string, payload []byte) error {
	token := mqtt.client.Publish(topic, 0, false, payload)

	// wait Token flow
	<-token.Done()
	return token.Error()
}

/*------------------------------------------------------------------------------*/ /*
	Publish Loop
*/ /*------------------------------------------------------------------------------*/
func (mqtt *MqttClient) publishLoop() {
	for {
		select {

		// Root canceling
		case <-mqtt.ctx.Done():
			close(mqtt.PublishCh)
			return

		// Publish Func
		case contents, isNotClose := <-mqtt.PublishCh:
			if !isNotClose {
				return
			}
			topic := fmt.Sprintf("%s/%s", mqtt.option.ClientID, contents.TopicPrefix)
			contents.Payload["TimeStamp"] = contents.TimeStamp.Format(time.RFC3339)
			payload, err := json.Marshal(contents.Payload)
			if err != nil {
				zap.S().Error("The marshalling process to JSON format failed. [", err, "]")
				zap.S().Debug(contents.Payload)
				continue
			}

			// Publish
			if err = retryhandler(defaultRetryCount, func() error {
				return mqtt.publishFn(topic, payload)
			}); err != nil {
				zap.S().Error(err)
			}
			zap.S().Debug(fmt.Sprintf("Publish Message!\n\t\t topic[%s] \n\t\t payload[%s]", topic, string(payload)))
		}
	}
}

/*------------------------------------------------------------------------------*/ /*
	Subscribe handler
*/ /*------------------------------------------------------------------------------*/
func (mqtt *MqttClient) subscribeFn(client MQTT.Client, contents MQTT.Message) {
	var payload map[string]interface{}
	if err := json.Unmarshal(contents.Payload(), &payload); err != nil {
		zap.S().Error(err)
		return
	}
	mqtt.SubscribeCh <- Contents{
		TimeStamp:   time.Now(),
		TopicPrefix: strings.Replace(contents.Topic(), mqtt.option.ClientID+"/", "", 1),
		Payload:     payload,
	}
}

/*------------------------------------------------------------------------------*/ /*
	Subscribe Loop (OneTopic)
*/ /*------------------------------------------------------------------------------*/
func (mqtt *MqttClient) subscribeLoop(topicPrefix string) {
	topic := fmt.Sprintf("%s/%s", mqtt.option.ClientID, topicPrefix)
	zap.S().Info("SubscribeTopic: ", topic)
	<-time.After(250 * time.Millisecond)
	for {
		token := mqtt.client.Subscribe(topic, 1, mqtt.subscribeFn)
		<-token.Done()
		if token.Error() != nil {
			zap.S().Error(token.Error())
			<-time.After(500 * time.Millisecond)
		}
	}
}

/*------------------------------------------------------------------------------*/ /*
	Subscribe Loop (MultipleTopic)
*/ /*------------------------------------------------------------------------------*/
func (mqtt *MqttClient) multipleSubscribeLoop(topicPrefixs []string) {
	topics := make(map[string]byte)
	for i, topicPrefix := range topicPrefixs {
		topics[fmt.Sprintf("%s/%s", mqtt.option.ClientID, topicPrefix)] = byte(i)
	}
	zap.S().Info("SubscribeTopics: ", topics)
	<-time.After(250 * time.Millisecond)
	for {
		token := mqtt.client.SubscribeMultiple(topics, mqtt.subscribeFn)
		<-token.Done()
		if token.Error() != nil {
			zap.S().Error(token.Error())
			<-time.After(500 * time.Millisecond)
		}
	}

}
