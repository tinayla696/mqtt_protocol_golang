package service

import (
	"context"
	"sync"
	"time"

	"github.com/tinayla696/mqtt_protocol_golang/service/task"
	"go.uber.org/zap"
)

// *--------------------------------------------------------------------------------------
// Worker
type Worker struct {
	id         int
	taskCh     <-chan task.Task
	quit       <-chan struct{}
	wg         *sync.WaitGroup
	workerType task.TaskType
}

// *--------------------------------------------------------------------------------------
// NewWorker (constructor)
func NewWorker(id int, taskCh <-chan task.Task, quit <-chan struct{}, wg *sync.WaitGroup, workerType task.TaskType) *Worker {
	return &Worker{
		id:         id,
		taskCh:     taskCh,
		quit:       quit,
		wg:         wg,
		workerType: workerType,
	}
}

// *--------------------------------------------------------------------------------------
// StartWorker
func (w *Worker) Start() {
	defer w.wg.Done()
	zap.S().Infof("Starting Woker Type %s, ID: %d started", w.workerType, w.id)
	for {
		select {
		case <-w.quit:
			zap.S().Infof("Worker %d quitting", w.id)
			return

		case task, ok := <-w.taskCh:
			if !ok {
				zap.S().Infof("Worker %d: task channel closed", w.id)
				return
			}
			zap.S().Debugf("Worker %d received task: %s", w.id, task.String())

			// Create a context for the task execution
			ctx, cancelFn := context.WithTimeout(context.Background(), 5000*time.Millisecond)
			err := task.Execute(ctx)
			cancelFn()

			if err != nil {
				zap.S().Errorf("Worker %d failed to execute task %s: %v", w.id, task.String(), err)
			}
		}
	}
}
