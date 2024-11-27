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
	for j := range jobs {
		Logger.Debug(
			"grid worker: processing cell",
			"grid_id", j.grid.Name,
			"cell_id", j.cell.Identifier,
			"mode", j.cell.mode,
		)
		if j.cell.mode == ModeOCR {
			Logger.Debug(
				"grid worker: starting ocr processing",
				"grid_id", j.grid.Name,
				"cell_id", j.cell.Identifier,
			)
			if err := j.cell.IdentifyOCR(); err != nil {
				j.res <- &Result{Ok: false, Error: fmt.Errorf("identifying ocr: %v", err)}
				return
			}
			Logger.Debug(
				"grid worker: finished ocr processing",
				"grid_id", j.grid.Name,
				"cell_id", j.cell.Identifier,
			)
		}

		if j.cell.mode == ModeComparison {
			Logger.Debug(
				"grid worker: starting value comparison",
				"grid_id", j.grid.Name,
				"cell_id", j.cell.Identifier,
			)
			if err := j.cell.ProcessValues(j.grid.digitComparisons); err != nil {
				j.res <- &Result{Ok: false, Error: fmt.Errorf("processing comparison values: %v", err)}
				return
			}
			Logger.Debug(
				"grid worker: finished value comparison",
				"grid_id", j.grid.Name,
				"cell_id", j.cell.Identifier,
			)

			Logger.Debug(
				"grid worker: starting placeholder comparison",
				"grid_id", j.grid.Name,
				"cell_id", j.cell.Identifier,
			)
			if err := j.cell.ProcessPlaceholders(j.grid.placeholderComparisons); err != nil {
				j.res <- &Result{Ok: false, Error: fmt.Errorf("processing placeholder values: %v", err)}
				return
			}
			Logger.Debug(
				"grid worker: finished placeholder comparison",
				"grid_id", j.grid.Name,
				"cell_id", j.cell.Identifier,
			)
		}

		j.res <- &Result{Ok: true}
	}
}

func (s *GridWorker) Start() {
	// my mac has 10 cores, but only 8 are performance,
	// running this at 10 causing the CPU to go to 800%
	// and the machine to slow down
	workers := 5

	Logger.Debug("starting grid worker", "workers", workers)

	for i := 0; i < 5; i++ {
		go s.work(s.jobs)
	}
}

func NewGridWorker() *GridWorker {
	return &GridWorker{
		jobs: make(chan *WorkerJob, 1000),
	}
}
