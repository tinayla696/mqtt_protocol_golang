package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tinayla696/mqtt_go/mqttd"
	"github.com/tinayla696/mqtt_go/service"
	"go.uber.org/zap"
)

func main() {
	clientID_Arg := flag.String("id", "", "MQTT ClientID")
	flag.Parse()

	/* Logger */
	logHandler := service.SetupLogService("../logs", zap.DebugLevel)
	defer logHandler.Logs.Sync()
	zap.ReplaceGlobals(logHandler.Logs)

	ctx, cancelFn := context.WithCancel(context.Background())

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	mqttConf := mqttd.MqttConf{
		Endpoint:              "127.0.0.1:1883",
		RootCA:                "",
		PrivateKey:            "",
		ClientCert:            "",
		SubscribeTopicPrefixs: []string{"sub/hoge", "sub/fuga"},
	}

	mqttH, err := mqttd.SetMqttClient(ctx, *clientID_Arg, mqttConf)
	if err != nil {
		panic(err)
	}
	mqttH.Start()

	defer func() {
		mqttH.End()
		cancelFn()
	}()

	func() {
		go generateMessage(ctx, &mqttH.PublishCh)
		go receiveMqttMsg(ctx, &mqttH.SubscribeCh)
	}()

	<-interrupt
}

func generateMessage(ctx context.Context, pubCh *chan mqttd.Contents) {
	for {
		select {
		case <-ctx.Done():
			return

		case <-time.After(10 * time.Second):
			*pubCh <- mqttd.Contents{
				TimeStamp:   time.Now(),
				TopicPrefix: "msg",
				Payload: map[string]interface{}{
					"timestamp": time.Now().Add(-1 * time.Minute),
					"status":    "on-line",
					"level":     99,
					"check":     true,
				},
			}
		}
	}
}

func receiveMqttMsg(ctx context.Context, subCh *chan mqttd.Contents) {
	for {
		select {
		case <-ctx.Done():
			return

		case contetns, isNotClose := <-*subCh:
			if !isNotClose {
				return
			}

			zap.S().Info(contetns)
		}
	}
}
