package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/tinayla696/mqtt_protocol_golang/develop"
	"github.com/tinayla696/mqtt_protocol_golang/module"
	"github.com/tinayla696/mqtt_protocol_golang/module/mqttm"
	"go.uber.org/zap"
)

const (
	ENV_PATH        string = "./conf.d/default.conf"
	KEY_DEVICE_ID   string = "DEVICE_ID"
	KEY_CONFIG_FILE string = "CONFIG_FILE"
	KEY_LOG_DIR     string = "LOG_DIR"
)

var (
	// Arguments
	isDebugArg *bool   = flag.Bool("debug", false, "Enable debug mode")
	envPathArg *string = flag.String("env", ENV_PATH, "Path to the environment configuration file")

	// Environment variables
	deviceIDEnv   string = "test00"
	configFileEnv string = "./conf.d/config.json"
	logDirEnv     string = "./log.d"

	// Global configuration
	conf            Config                   = Config{}
	mqttConnections map[string]*mqttm.Module = make(map[string]*mqttm.Module)
)

type (
	Config struct {
		MQTT map[string]mqttm.Config `json:"MQTT"`
	}
)

func main() {
	// Parse command line arguments
	flag.Parse()

	// Load environment variables
	if err := godotenv.Load(*envPathArg); err != nil {
		panic("Error loading environment variables: " + err.Error())
	}
	deviceIDEnv = os.Getenv(KEY_DEVICE_ID)
	configFileEnv = os.Getenv(KEY_CONFIG_FILE)
	logDirEnv = os.Getenv(KEY_LOG_DIR)

	// Setup Logging
	loggerModule := module.SetLogger(logDirEnv, *isDebugArg)
	defer loggerModule.Sync()
	zap.ReplaceGlobals(loggerModule)

	// Development mode
	if *isDebugArg {
		zap.S().Info("Debug mode is enabled")
		prof, err := develop.ProfileInit(logDirEnv)
		if err != nil {
			zap.S().Fatal("Failed to initialize profiling", zap.Error(err))
		}
		defer prof.Stop()
		zap.S().Info("Profiling initialized", zap.String("logDir", logDirEnv))
	}

	// Load configuration
	if confd, err := os.ReadFile(configFileEnv); err != nil {
		zap.S().Panicf("Failed to read configuration file: %s", err.Error())
	} else {
		if err := json.Unmarshal(confd, &conf); err != nil {
			zap.S().DPanic("Failed to parse configuration file", zap.Error(err))
		}
	}
	zap.S().Debugf("Configuration loaded successfully \n %+v", conf)

	// Context & Interrupt handling
	ctx, cancelFn := context.WithCancel(context.Background())
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Setup MQTT Module
	for hostName, mqttConf := range conf.MQTT {
		mqttModule, err := mqttm.New(ctx, deviceIDEnv, hostName, mqttConf)
		if err != nil {
			zap.S().Fatalf("Failed to create MQTT module for %s: %s", hostName, err.Error())
			continue
		}
		if err := mqttModule.Run(); err != nil {
			zap.S().Fatalf("Failed to run MQTT module for %s: %s", hostName, err.Error())
			continue
		}
		mqttConnections[hostName] = mqttModule
		zap.S().Infof("MQTT module for %s is running", hostName)
	}

	// Handle interrupt signal
	<-interrupt
	zap.S().Info("Received interrupt signal, shutting down...")
	for hostName, mqttModule := range mqttConnections {
		mqttModule.Stop()
		zap.S().Infof("MQTT module for %s has been stopped", hostName)
	}
	cancelFn()

}
