package jobs

import (
	"context"
	"fmt"

	"app/pkg/queue"

	"github.com/hibiken/asynq"
)

// RegisterAsynqHandlers registers task handlers on mux (task type = job_type JSON / PushRaw task_type).
func RegisterAsynqHandlers(mux *asynq.ServeMux) {
	if mux == nil {
		return
	}
	register := func(typeName string) {
		tn := typeName
		mux.HandleFunc(tn, func(ctx context.Context, task *asynq.Task) error {
			job, err := queue.DecodeJobFromJSON(task.Payload())
			if err != nil {
				return fmt.Errorf("decode job %q: %w", tn, err)
			}
			return job.Handle()
		})
	}
	register(JobTypeExample)
	register(JobTypeSendWelcomeEmail)
	register(JobTypeProcessUpload)
	register(JobTypeCleanup)
	register(JobTypeProcessOrder)
	mux.HandleFunc("raw", func(ctx context.Context, task *asynq.Task) error {
		_ = ctx
		_ = task
		return nil
	})
}
