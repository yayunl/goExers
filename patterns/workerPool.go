package patterns

import (
	"context"
	"fmt"
	"sync"
)

type Result struct {
	JobID       interface{}
	Value       interface{}
	Err         error
	Description string
}

// Tasks
type Tasker interface {
	Execute(args []any) Result
}

// Job implements `Tasker` interface
type Job struct {
	ID   int
	Fn   func(arg []any) Result
	Args []any
}

func (t Job) Execute(args []any) Result {
	result := t.Fn(args)
	return result
}

func ExecuteTask(t Tasker, args []any) Result {
	return t.Execute(args)
}

// WorkerPool
type WorkerPool struct {
	MaxWorkers   int
	JobStream    <-chan Job
	ResultStream chan Result
	Done         chan struct{}
}

func New(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		MaxWorkers:   maxWorkers,
		JobStream:    make(<-chan Job, 255),
		ResultStream: make(chan Result, 255),
		Done:         make(chan struct{}),
	}
}

func (wp *WorkerPool) Run(ctx context.Context) {
	var wg sync.WaitGroup

	// Fan out worker goroutines
	for i := 0; i < wp.MaxWorkers; i++ {
		wg.Add(1)
		go func(workerID int, ctx context.Context, wg *sync.WaitGroup, jobStream <-chan Job, resultStream chan<- Result) {
			defer wg.Done()
			for {
				select {
				case task, ok := <-jobStream:
					if !ok {
						// Jobs channel is closed
						return
					}
					// execute the job and gather results
					wp.ResultStream <- ExecuteTask(task, task.Args)
				case <-ctx.Done():
					value := fmt.Sprintf("cancelled worker %d. Error detail: %v\n", workerID, ctx.Err())
					resultStream <- Result{
						Value: value,
						Err:   ctx.Err(),
					}
					return
				}
			}
		}(i, ctx, &wg, wp.JobStream, wp.ResultStream)
	}

	// Fan-in results
	wg.Wait()
	close(wp.ResultStream)
}

func (wp *WorkerPool) Results() <-chan Result {
	return wp.ResultStream
}

func (wp *WorkerPool) Assign(jobs <-chan Job) {
	wp.JobStream = jobs
}

func JobGenerator(done <-chan struct{}, tasks ...Job) <-chan Job {
	dataStream := make(chan Job)
	// a pipeline stage for creating jobs
	go func() {
		defer close(dataStream)
		for _, task := range tasks {
			select {
			case <-done:
				return
			case dataStream <- task:
			}
		}
	}()
	return dataStream
}
