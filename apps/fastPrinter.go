package apps

import (
	"context"
	"fmt"
	"goExers/patterns"
	"strconv"
	"time"
)

func createPrintTasks(pitems []string) []patterns.Job {
	var tasks []patterns.Job
	for pid, item := range pitems {
		newTask := patterns.Job{
			ID: pid,
			Fn: func(args []any) patterns.Result {
				time.Sleep(time.Second)
				return patterns.Result{
					JobID:       args[0],
					Value:       fmt.Sprintf("Print task ID: %s, Printed item: %s\n", args[0], args[1]),
					Err:         nil,
					Description: fmt.Sprintf("Print task ID: %s", args[0]),
				}
			},
			Args: []any{strconv.Itoa(pid), item},
		}
		tasks = append(tasks, newTask)
	}

	return tasks
}

func FastPrinter(maxWorkers int) {
	done := make(chan struct{})
	defer close(done)
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	// Create a batch of tasks
	var printTasks []string
	for i := 0; i < 100; i++ {
		printTasks = append(printTasks, fmt.Sprintf("page %d", i))
	}

	tasks := createPrintTasks(printTasks)
	wp := patterns.New(maxWorkers)
	taskStream := patterns.JobGenerator(done, tasks...)
	wp.Assign(taskStream)

	// Start workers
	go wp.Run(ctx)
	// Collect results
	results := wp.Results()

	for result := range results {
		fmt.Printf("Job ID=> %s, Task=> %s\n", result.JobID, result.Value)
	}
}
