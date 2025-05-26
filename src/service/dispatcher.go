package service

import (
	"fmt"
	"sync"

	"github.com/tinayla696/mqtt_protocol_golang/module/mqttm"
	"github.com/tinayla696/mqtt_protocol_golang/service/task"
	"go.uber.org/zap"
)

// *---------------------------------------------------------------------------------------------------------------------------------
// Dispatcher
type Dispatcher struct {
	// Input Channels
	MqttClients map[string]*mqttm.Module // MQTT Clients

	// Workers Queue
	mqttTaskQue chan task.Task

	quit     chan struct{}
	wg       *sync.WaitGroup // Dispatcherと結果PorcessのWaitGroup
	workerWg *sync.WaitGroup // Worker全体のWaitGroup

	// Worker Type
	numMqttWorkers int

	nextTaskID int // 次のタスクID
}

// *--------------------------------------------------------------------------------------------------
// NewDispatcher (constructor)
func NewDispatcher(mqttClients map[string]*mqttm.Module, mqttWorkers int) *Dispatcher {
	return &Dispatcher{
		MqttClients:    mqttClients,
		mqttTaskQue:    make(chan task.Task, mqttWorkers),
		quit:           make(chan struct{}),
		wg:             &sync.WaitGroup{},
		workerWg:       &sync.WaitGroup{},
		numMqttWorkers: mqttWorkers,
		nextTaskID:     0,
	}
}

// *--------------------------------------------------------------------------------------------------
// Start
func (d *Dispatcher) Start() {
	zap.S().Info("Starting Dispatcher...")
	d.wg.Add(1)

	// DEV: 各Workerを起動
	d.launchWorkers(d.numMqttWorkers, d.mqttTaskQue, task.MqttTaskType)

	// MQTT Subscription Loop
	for domain, client := range d.MqttClients {
		go d.monitorMqttSubscription(domain, client)
	}
}

// *--------------------------------------------------------------------------------------------------
// lanchWorkers
func (d *Dispatcher) launchWorkers(numWorkers int, taskCh <-chan task.Task, workerType task.TaskType) {
	for i := 0; i < numWorkers; i++ {
		d.workerWg.Add(1)
		w := NewWorker(i+1, taskCh, d.quit, d.workerWg, workerType)
		go w.Start()
	}
}

// *--------------------------------------------------------------------------------------------------
// monitorMqttSubscription
func (d *Dispatcher) monitorMqttSubscription(hostname string, client *mqttm.Module) {
	defer d.wg.Done()
	zap.S().Infof("Monitoring MQTT subscription on channel: %s", hostname)
	for {
		select {
		case <-d.quit:
			zap.S().Warn("Dispatcher received quit signal, stopping MQTT subscription monitoring")
			return

		case subContents, ok := <-client.SubCh:
			if !ok {
				zap.S().Warn("MQTT subscription channel closed, stopping monitoring")
				return
			}
			d.nextTaskID++
			taskContents := &task.MqttTask{
				Contents: subContents,
				ID:       d.nextTaskID,
			}
			if err := d.assignTaskToQue(taskContents, d.mqttTaskQue, task.MqttTaskType); err != nil {
				zap.S().Errorf("Failed to assign task to queue: %v", err)
			}
		}
	}
}

// *--------------------------------------------------------------------------------------------------
// assignTaskToQue
func (d *Dispatcher) assignTaskToQue(task task.Task, queue chan task.Task, expectedType task.TaskType) error {
	if task.Type() != expectedType {
		return fmt.Errorf("task type mismatch: expected %s, got %s", expectedType, task.Type())
	}
	select {
	case queue <- task:
		zap.S().Debug("Task assigned to queue:", zap.String("task", task.String()))
		return nil
	case <-d.quit:
		return fmt.Errorf("dispatcher is quitting, task %s not assigned", task.String())
	default:
		return fmt.Errorf("task queue is full, task %s not assigned", task.String())
	}
}

// *--------------------------------------------------------------------------------------------------
// Stop
func (d *Dispatcher) Stop() {
	zap.S().Info("Stopping Dispatcher...")
	// DEV: 1. Stop all workers
	close(d.quit)

	// DEV: 各タスクキューを閉じる
	close(d.mqttTaskQue) // Close the task queue

	// DEV: すべてのWorkerの終了待ち
	d.workerWg.Wait() // Wait for all workers to finish

	// DEV: タスク振り分けのルーチンと結果プロセスの終了待機
	d.wg.Wait() // Wait for all goroutines to finish
	zap.S().Info("Dispatcher stopped successfully")
}
