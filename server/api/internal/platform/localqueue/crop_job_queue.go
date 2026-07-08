package localqueue

import (
	"context"
	"sync"

	"github.com/tomy/guess-the-celebrity/server/api/internal/module/job"
)

type CropJobQueue struct {
	mu   sync.Mutex
	jobs []job.CropJob
}

func NewCropJobQueue() *CropJobQueue {
	return &CropJobQueue{}
}

func (q *CropJobQueue) EnqueueCropJob(_ context.Context, cropJob job.CropJob) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.jobs = append(q.jobs, cropJob)
	return nil
}

func (q *CropJobQueue) Jobs() []job.CropJob {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]job.CropJob(nil), q.jobs...)
}
