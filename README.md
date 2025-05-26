# Golang MQTT Protocol

This project is a Golang-based implementation for testing MQTT communication.  
It provides a modular structure for managing MQTT clients, tasks, and workers, along with logging and profiling capabilities.

## Features

- **MQTT Communication**: Implements MQTT client functionality using the [Paho MQTT library](https://github.com/eclipse/paho.mqtt.golang).
- **Task Management**: Supports task-based processing with a dispatcher and worker model.
- **Logging**: Configurable logging with support for production, development, and test modes.
- **Profiling**: CPU and memory profiling for debugging and performance analysis.
- **Configuration**: JSON-based configuration for MQTT clients and environment variables.

## Project Structure

``` plaintext
.gitignore
README.md
conf.d/
    config.json
    default.conf
log.d/
    ***.log
    prof/
        cpu.prof
        mem.prof
src/
    go.mod
    go.sum
    main.go
    develop/
        profile.go
    module/
        logger.go
        mqttm/
            connect.go
            handler.go
            mqtt.go
            publish.go
            subscribe.go
    service/
        dispatcher.go
        worker.go
        task/
            common.go
            mqtt_task.go
```

### Key Components

- **`src/main.go`**: Entry point of the application. Initializes logging, loads configuration, sets up MQTT clients, and starts the dispatcher.
- **`src/module/mqttm/`**: Contains the MQTT module implementation, including connection handling, publishing, and subscribing.
- **`src/service/`**: Implements the dispatcher and worker model for task processing.
- **`src/develop/profile.go`**: Provides profiling utilities for CPU and memory usage.
- **`src/module/logger.go`**: Configures the logging system with support for different modes.

## Getting Started

### Prerequisites

- Go 1.23.3 or later
- MQTT broker (e.g., Mosquitto)

### Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/tinayla696/mqtt_protocol_golang.git
   cd mqtt_protocol_golang/src
   ```

2. Install dependencies:

   ```bash
   go mod tidy
   ```

3. Configure the environment:
   - Update `conf.d/config.json` with your MQTT broker details.
   - Set environment variables in `conf.d/default.conf`.

### Running the Application

1. Start the application in production mode:

   ```bash
   go run main.go -m proc
   ```

2. Start the application in debug mode (with profiling):

   ```bash
   go run main.go -m debug
   ```

### Configuration

The application uses a JSON configuration file (`conf.d/config.json`) to define MQTT client settings. Example:

```json
{
  "MQTT": {
    "broker1": {
      "endpoint": "mqtt.example.com",
      "username": "user",
      "password": "pass",
      "root_ca": "path/to/ca.pem",
      "key": "path/to/key.pem",
      "cert": "path/to/cert.pem",
      "subscribe_topics": {
        "topic1": 0,
        "topic2": 1
      }
    }
  }
}
```

### Logging

Logs are stored in the `log.d/` directory. The application automatically rotates logs when the maximum number of files is reached. Old log files are deleted to maintain the limit.

### Profiling

Profiling is enabled in debug mode. CPU and memory profiles are saved in the `log.d/prof/` directory as `cpu.prof` and `mem.prof`.

## License

This project is licensed under the MIT License. See the LICENSE file for details.

## Acknowledgments

- [Paho MQTT Golang](https://github.com/eclipse/paho.mqtt.golang)
- [Zap Logging](https://github.com/uber-go/zap)
- [godotenv](https://github.com/joho/godotenv)
