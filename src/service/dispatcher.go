package service

import (
	"context"
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
	taskQue chan task.Task

	// quit     chan struct{}
	ctx      context.Context
	cancelFn context.CancelFunc // コンテキストのキャンセル関数
	wg       *sync.WaitGroup    // Dispatcherと結果PorcessのWaitGroup
	workerWg *sync.WaitGroup    // Worker全体のWaitGroup

	// Worker Type
	numMqttWorkers int

	nextTaskID int // 次のタスクID
}

// *--------------------------------------------------------------------------------------------------
// NewDispatcher (constructor)
func NewDispatcher(parentCtx context.Context, mqttClients map[string]*mqttm.Module, mqttWorkers int) *Dispatcher {
	ctx, cancelFn := context.WithCancel(parentCtx) // コンテキストのキャンセル関数を作成
	return &Dispatcher{
		MqttClients:    mqttClients,
		taskQue:        make(chan task.Task, mqttWorkers),
		ctx:            ctx,
		cancelFn:       cancelFn,
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

	// DEV: 各Workerを起動
	if len(d.MqttClients) > 0 {
		d.launchWorkers(d.numMqttWorkers, d.taskQue, task.MqttTaskType)
	}

	// MQTT Subscription Loop
	for domain, client := range d.MqttClients {
		d.wg.Add(1)
		go d.monitorMqttSubscription(domain, client)
	}
}

// *--------------------------------------------------------------------------------------------------
// lanchWorkers
func (d *Dispatcher) launchWorkers(numWorkers int, taskCh <-chan task.Task, workerType task.TaskType) {
	for i := 0; i < numWorkers; i++ {
		d.workerWg.Add(1)
		w := NewWorker(i+1, taskCh, d.ctx.Done(), d.workerWg, workerType)
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
		case <-d.ctx.Done():
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
			if err := d.assignTaskToQue(taskContents, d.taskQue, task.MqttTaskType); err != nil {
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
	case <-d.ctx.Done():
		return fmt.Errorf("dispatcher is quitting, task %s not assigned", task.String())
	default:
		return fmt.Errorf("task queue is full, task %s not assigned", task.String())
	}
}

// *--------------------------------------------------------------------------------------------------
// Stop
func (d *Dispatcher) Stop() {
	zap.S().Info("Stopping Dispatcher...")
	// DEV: No.1 コンテキストをキャンセルして、全てののGoroutineを停止
	d.cancelFn()

	// DEV: No.2 taskQueに書き込むProducer Goroutineを全て終了する
	d.wg.Wait()

	// DEV: No.3 taskQueを閉じる
	close(d.taskQue)

	// DEV: No.4 Worker全体の終了待機
	d.workerWg.Wait()
	zap.S().Info("Dispatcher stopped successfully")
}
