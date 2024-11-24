package service

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// public

	// private
	_MAXLOGFILE int = 2
)

/*------------------------------------*/ /*
	Class
*/ /*------------------------------------*/
type LogService struct {
	Logs        *zap.Logger
	LogDir      string
	LogFilePath string
}

/*------------------------------------*/ /*
	Constructor
*/ /*------------------------------------*/
func SetupLogService(path string, logLevel zapcore.Level) *LogService {
	myself := new(LogService)
	myself.LogDir = path

	// check Log dir
	var logFlag bool
	if fs, err := os.Stat(myself.LogDir); os.IsNotExist(err) || !fs.IsDir() {
		fmt.Printf("%v\n", "There is no log save destiantion")

		// make log dir
		if err := os.Mkdir(myself.LogDir, 0777); err != nil {
			fmt.Printf("%v\n", "Unable to create save destintion...................")
			logFlag = false

			// select continue
			if logLevel == zap.DebugLevel {
				scan := bufio.NewScanner(os.Stdin)
				fmt.Printf("%v\n", "Will you continue? [Y/-]")
				fmt.Print("> ")
				scan.Scan()
				if scan.Text() == "Y" {
					fmt.Printf("%v\n", "Operation without log save............")
					logFlag = true
				}
			}

			// Continue
			if !logFlag {
				fmt.Printf("%v\n %v\n", "Check the log directory and restart.........", "Goodbye....")
				os.Exit(2)
			}
		}
	}

	// check count of log files
	files, _ := os.ReadDir(myself.LogDir)
	if len(files) > _MAXLOGFILE {
		fmt.Printf("Max log file %v\n", _MAXLOGFILE)

		var lastTime time.Time
		var deleteFile string
		skipFlag := true
		for _, fileProp := range files {
			fileInfo, _ := fileProp.Info()
			if skipFlag {
				lastTime = fileInfo.ModTime()
				deleteFile = fileInfo.Name()
				skipFlag = false
				continue
			}

			if lastTime.After(fileInfo.ModTime()) {
				lastTime = fileInfo.ModTime()
				deleteFile = fileInfo.Name()
			}
		}

		// delete oldest file
		if deleteFile != "" {
			deleteFilePath := filepath.Join(myself.LogDir, deleteFile)
			os.Remove(deleteFilePath)
			fmt.Printf("Delete logfilepath: %v\n", deleteFilePath)
		}
	}

	// generate log file
	myself.LogFilePath = filepath.Join(myself.LogDir, time.Now().Format("20060102-150405")+".log")

	// setup zap config
	logConfig := generateZapLogConfig(logLevel)
	if logLevel == zapcore.DebugLevel {
		logConfig.OutputPaths = append(logConfig.OutputPaths, myself.LogFilePath)
	} else {
		logConfig.OutputPaths = []string{myself.LogFilePath}
	}

	// build zap logger
	buildLog, err := logConfig.Build()
	if err != nil {
		panic(err)
	}
	myself.Logs = buildLog
	return myself
}

/*-------------------------------------------------------*/ /*
	Generate zap log config
*/ /*-------------------------------------------------------*/
func generateZapLogConfig(logLevel zapcore.Level) zap.Config {
	logConfig := zap.NewDevelopmentConfig()
	logConfig.Level = zap.NewAtomicLevelAt(logLevel)
	logConfig.DisableStacktrace = false
	logConfig.DisableCaller = true
	logConfig.Development = false
	logConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	if logLevel == zapcore.DebugLevel {
		logConfig.DisableStacktrace = true
		logConfig.DisableCaller = false
		logConfig.Development = true
	}

	return logConfig
}
