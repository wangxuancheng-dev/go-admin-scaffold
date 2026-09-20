package jobs

import "go-admin-scaffold/pkg/queue"

// 队列 job_type 常量（与 RegisterJobType 一致）
const (
	JobTypeExample          = "example"
	JobTypeSendWelcomeEmail = "send_welcome_email"
	JobTypeProcessUpload    = "process_upload"
	JobTypeCleanup          = "cleanup"
	JobTypeProcessOrder     = "process_order"
)

func init() {
	queue.RegisterJobType(JobTypeExample, func() queue.JobInterface { return &ExampleJob{} })
	queue.RegisterJobType(JobTypeSendWelcomeEmail, func() queue.JobInterface { return &SendWelcomeEmailJob{} })
	queue.RegisterJobType(JobTypeProcessUpload, func() queue.JobInterface { return &ProcessUploadJob{} })
	queue.RegisterJobType(JobTypeCleanup, func() queue.JobInterface { return &CleanupJob{} })
	queue.RegisterJobType(JobTypeProcessOrder, func() queue.JobInterface { return &ProcessOrderJob{} })
}
