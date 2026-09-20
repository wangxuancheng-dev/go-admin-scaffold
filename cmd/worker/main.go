package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/core/jobs"
	"go-admin-scaffold/pkg/queue"

	"github.com/hibiken/asynq"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if len(cfg.Queue.Queues) == 0 {
		log.Fatal("no queues configured (queue.queues in config)")
	}

	redisURL := config.QueueRedisConnectionURL(cfg)
	redisOpt, err := queue.AsynqRedisConnOpt(redisURL)
	if err != nil {
		log.Fatalf("redis url: %v", err)
	}

	queues := make(map[string]int)
	concurrency := 0
	for name, qd := range cfg.Queue.Queues {
		p := qd.Priority
		if p <= 0 {
			p = 1
		}
		queues[name] = p
		proc := qd.Processes
		if proc < 1 {
			proc = 1
		}
		concurrency += proc
	}
	if concurrency <= 0 {
		concurrency = 1
	}

	mux := asynq.NewServeMux()
	jobs.RegisterAsynqHandlers(mux)

	shutdownSec := cfg.Queue.Worker.Timeout
	if shutdownSec <= 0 {
		shutdownSec = 30
	}
	srv := asynq.NewServer(redisOpt, asynq.Config{
		Concurrency:     concurrency,
		Queues:          queues,
		ShutdownTimeout: time.Duration(shutdownSec) * time.Second,
	})

	log.Printf("asynq worker: redis=%s concurrency=%d queues=%v", redisURL, concurrency, queues)

	// 使用 Start + 本进程单独 Notify + Shutdown：避免与 Run 内 waitForSignals 抢同一信号（曾导致无法退出），
	// 且在 Windows 上 Ctrl+C 后尽量走 Shutdown 正常 return，减少 exit status 0xc000013a（CONTROL_C_EXIT）。
	if err := srv.Start(mux); err != nil {
		log.Fatalf("asynq start: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	log.Println("worker running; Ctrl+C or SIGTERM to stop")
	<-sigCh
	signal.Stop(sigCh)

	log.Println("shutting down...")
	srv.Shutdown()
	log.Println("asynq server exited")
}
