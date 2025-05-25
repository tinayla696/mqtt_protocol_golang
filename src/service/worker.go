package service

import "sync"

type Task interface {
	Process()
}

type Worker struct {
	workerPool chan chan Task
	TaskCh     chan Task
}

func NewWorker(workerPool chan chan Task) *Worker {
	return &Worker{
		workerPool: workerPool,
		TaskCh:     make(chan Task),
	}
}

func (w *Worker) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		w.workerPool <- w.TaskCh
		task := <-w.TaskCh
		task.Process()
	}
}
