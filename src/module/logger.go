package module

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	isSkip bool = true
)

const (
	MAX_LOG_FILES = 3
)

func SetLogger(path string, isDebug bool) *zap.Logger {
	if fs, err := os.Stat(path); os.IsNotExist(err) || !fs.IsDir() {
		log.Fatalf("log path %s is not a directory", path)

		// Create log directory if it does not exist
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			log.Fatalf("failed to create log directory %s: %v", path, err)

			if isDebug {
				scan := bufio.NewScanner(os.Stdin)
				fmt.Print("Do you want to continue? (y/n): ")
				scan.Scan()
				if scan.Text() != "y" {
					log.Fatal("Exiting due to log directory creation failure")
					os.Exit(1)
				}
				log.Println("Continuing without logging")
			}
		}
	}

	// Ckeck if the log file exists
	files, _ := os.ReadDir(path)
	if len(files) >= MAX_LOG_FILES {
		log.Printf("Log directory %s has reached the maximum number of files (%d). Skipping logging.", path, MAX_LOG_FILES)
		isSkip = true
		var lastTime time.Time
		var deleteFile string
		for _, fileProp := range files {
			if fileProp.IsDir() {
				continue
			}

			fileInfo, _ := fileProp.Info()
			if isSkip {
				lastTime = fileInfo.ModTime()
				deleteFile = fileInfo.Name()
				isSkip = false
				continue
			}

			if lastTime.After(fileInfo.ModTime()) {
				lastTime = fileInfo.ModTime()
				deleteFile = fileInfo.Name()
			}
		}

		// Delete log file
		if deleteFile != "" {
			deleteFilePath := filepath.Join(path, deleteFile)
			os.Remove(deleteFilePath)
			log.Printf("Deleted old log file: %s", deleteFilePath)
		}
	}

	// Setup logger
	var logConf zap.Config
	logPath := filepath.Join(path, time.Now().Format("20060102-150405")+".log")
	if isDebug {
		// Development mode
		log.Println("Setting up logger in development mode")
		logConf = zap.NewDevelopmentConfig()
		logConf.Encoding = "console"
		logConf.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logConf.OutputPaths = append(logConf.OutputPaths, logPath)
	} else {
		// Production mode
		log.Println("Setting up logger in production mode")
		logConf = zap.NewProductionConfig()
		logConf.Encoding = "json"
		logConf.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		logConf.OutputPaths = append(logConf.OutputPaths, logPath)
	}

	// Build the logger
	buildLog, err := logConf.Build()
	if err != nil {
		log.Fatalf("failed to build logger: %v", err)
		os.Exit(1)
	}
	return buildLog
}
