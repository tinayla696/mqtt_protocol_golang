package service

import (
	"fmt"
	"sync"

	"github.com/tinayla696/mqtt_protocol_golang/module/mqttm"
	"github.com/tinayla696/mqtt_protocol_golang/service/process"
)

type Dispatcher struct {
	workerPool chan chan Task
	taskQueue  chan Task
	shutdown   chan struct{}
	wg         sync.WaitGroup
}

func Newdispatcher(maxWorkers int) *Dispatcher {
	workerPool := make(chan chan Task, maxWorkers)
	return &Dispatcher{
		workerPool: workerPool,
		taskQueue:  make(chan Task), // Adjust buffer size as needed
		shutdown:   make(chan struct{}),
	}
}

func (d *Dispatcher) Run() {
	for i := 0; i < cap(d.workerPool); i++ {
		w := NewWorker(d.workerPool)
		d.wg.Add(1)
		go w.Start(&d.wg)
	}

	go d.dispatch()
}

func (d *Dispatcher) Start(subCh chan mqttm.Contents) {
	go func() {
		for data := range subCh {
			task := &process.MqttProcessingTask{Contents: data}
			d.EnqueueTask(task)
		}
	}()
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case task := <-d.taskQueue:
			workerCh := <-d.workerPool
			workerCh <- task
		case <-d.shutdown:
			fmt.Println("Dispatcher shutting down")
			return
		}
	}
}

func (d *Dispatcher) EnqueueTask(task Task) {
	d.taskQueue <- task
}

func (d *Dispatcher) Stop() {
	close(d.shutdown)
	d.wg.Wait()
	fmt.Println("All workers have been stopped")
}
