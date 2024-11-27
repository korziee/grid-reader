package internal

import (
	"fmt"
)

type GridWorker struct {
	jobs chan *WorkerJob
}

type Result struct {
	Ok    bool
	Error error
}

type WorkerJob struct {
	cell *Cell
	grid *Grid

	res chan<- *Result
}

func (s *GridWorker) work(jobs <-chan *WorkerJob) {
	// todo: debug log these jobs, give jobs an ID and follow their completii
	for j := range jobs {
		if j.cell.mode == ModeOCR {
			if err := j.cell.IdentifyOCR(); err != nil {
				j.res <- &Result{Ok: false, Error: fmt.Errorf("identifying ocr: %v", err)}
				return
			}
		}

		if j.cell.mode == ModeComparison {
			if err := j.cell.ProcessValues(j.grid.digitComparisons); err != nil {
				j.res <- &Result{Ok: false, Error: fmt.Errorf("processing comparison values: %v", err)}
				return
			}

			if err := j.cell.ProcessPlaceholders(j.grid.placeholderComparisons); err != nil {
				j.res <- &Result{Ok: false, Error: fmt.Errorf("processing placeholder values: %v", err)}
				return
			}
		}

		j.res <- &Result{Ok: true}
	}
}

func (s *GridWorker) Start() {
	// my mac has 10 cores, but only 8 are performance,
	// running this at 10 causing the CPU to go to 800%
	// and the machine to slow down
	for i := 0; i < 5; i++ {
		go s.work(s.jobs)
	}
}

func NewGridWorker() *GridWorker {
	return &GridWorker{
		jobs: make(chan *WorkerJob, 1000),
	}
}
