package tools

import (
	"sync"
)

// RunWithWorkers runs a job function over a slice of inputs with controlled concurrency.
func RunWithWorkers[T any](jobs []T, maxWorkers int, handler func(T)) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxWorkers)

	for _, job := range jobs {
		wg.Add(1)
		sem <- struct{}{}

		go func(j T) {
			defer wg.Done()
			defer func() { <-sem }()

			handler(j)
		}(job)
	}

	wg.Wait()
}
