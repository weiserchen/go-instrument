package processor

func NewParallelProcessor(worker int) *ParallelProcessor {
	return &ParallelProcessor{
		worker: worker,
	}
}

type ParallelProcessor struct {
	worker int
}
