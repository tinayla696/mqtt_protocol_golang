package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/tinayla696/mqtt_protocol_golang/develop"
	"github.com/tinayla696/mqtt_protocol_golang/module"
	"github.com/tinayla696/mqtt_protocol_golang/module/mqttm"
	"github.com/tinayla696/mqtt_protocol_golang/service"
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
	appModeArg *string = flag.String("m", "proc", "Mode of operation (proc/debug/test)")
	envPathArg *string = flag.String("env", ENV_PATH, "Path to the environment configuration file")

	// Environment variables
	deviceIDEnv   string = "test00"
	configFileEnv string = "./conf.d/config.json"
	logDirEnv     string = "./log.d"

	// Global configuration
	conf        Config                   = Config{}
	mqttClients map[string]*mqttm.Module = make(map[string]*mqttm.Module)
)

type (
	Config struct {
		MQTT map[string]*mqttm.Config `json:"MQTT"`
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
	loggerModule := module.SetLogger(logDirEnv, *appModeArg)
	defer loggerModule.Sync()
	zap.ReplaceGlobals(loggerModule)

	// Development mode
	if *appModeArg != "proc" {
		zap.S().Info("Debug mode is enabled")
		prof, err := develop.ProfileInit(filepath.Join(logDirEnv, "prof"))
		if err != nil {
			zap.S().Fatalf("Failed to initialize profiler: %v", err)
		}
		defer prof.Stop()
		zap.S().Infof("Profiling initialized %s", logDirEnv)
	}

	// Load configuration
	if confd, err := os.ReadFile(configFileEnv); err != nil {
		zap.S().Fatalf("Failed to read configuration file: %s", err.Error())
	} else {
		if err := json.Unmarshal(confd, &conf); err != nil {
			zap.S().Fatalf("Failed to parse configuration file %v", err)
		}
	}
	// zap.S().Debugf("Configuration loaded successfully \n %+v", conf)

	// Context & Interrupt handling
	ctx, cancelFn := context.WithCancel(context.Background())
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Setup MQTT Module
	for hostName, mqttConf := range conf.MQTT {
		mqttModule, err := mqttm.New(ctx, deviceIDEnv, hostName, *mqttConf)
		if err != nil {
			zap.S().Warnf("Failed to create MQTT module for %s: %v", hostName, err)
			continue
		}
		if err := mqttModule.Run(); err != nil {
			zap.S().Warnf("Failed to run MQTT module for %s: %v", hostName, err)
			continue
		}
		mqttClients[hostName] = mqttModule
		zap.S().Infof("MQTT module for %s is running", hostName)
	}

	// Start Dispatcher / Worker
	dw := service.NewDispatcher(mqttClients, 1)
	dw.Start()

	// Handle interrupt signal
	<-interrupt
	zap.S().Info("Received interrupt signal, shutting down...")
	for hostName, mqttModule := range mqttClients {
		mqttModule.Stop()
		zap.S().Infof("MQTT module for %s has been stopped", hostName)
	}
	cancelFn()

	dw.Stop() // Stop the dispatcher and wait for workers to finish

}
