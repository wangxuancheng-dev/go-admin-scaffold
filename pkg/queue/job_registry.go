package queue

import (
	"encoding/json"
	"reflect"
	"sync"
)

var (
	jobRegMu       sync.RWMutex
	jobFactories   = make(map[string]func() JobInterface)
	jobTypeByType  = make(map[reflect.Type]string) // 具体类型 -> job_type（RegisterJobType 时写入）
)

// RegisterJobType 注册任务类型，供 Asynq 消费端按 JSON 字段 job_type 反序列化为具体类型并调用 Handle。
// name 应稳定（如 "example"）；factory 每次返回新的零值指针，例如 func() JobInterface { return &ExampleJob{} }。
// 通常在 init 中注册，且业务进程需 import 该包以执行 init（如 _ "app/internal/core/jobs"）。
func RegisterJobType(name string, factory func() JobInterface) {
	if name == "" || factory == nil {
		return
	}
	jobRegMu.Lock()
	defer jobRegMu.Unlock()
	jobFactories[name] = factory
	sample := factory()
	t := concreteElemType(reflect.TypeOf(sample))
	if t != nil {
		jobTypeByType[t] = name
	}
}

func concreteElemType(t reflect.Type) reflect.Type {
	if t == nil {
		return nil
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// ensureJobTypeForPush 若嵌入的 BaseJob.JobType 为空，则按已注册的具体类型自动填入 job_type。
func ensureJobTypeForPush(job JobInterface) {
	if job == nil {
		return
	}
	bp := embeddedBaseJobPtr(job)
	if bp == nil || bp.JobType != "" {
		return
	}
	jobRegMu.RLock()
	defer jobRegMu.RUnlock()
	t := concreteElemType(reflect.TypeOf(job))
	if t == nil {
		return
	}
	if name, ok := jobTypeByType[t]; ok {
		bp.JobType = name
	}
}

// DecodeJobFromJSON 根据 job_type 解码为已注册的具体类型，否则为 *BaseJob；均会 hydrate Payload。
func DecodeJobFromJSON(raw []byte) (JobInterface, error) {
	var probe struct {
		JobType string `json:"job_type"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, err
	}
	jobRegMu.RLock()
	factory, ok := jobFactories[probe.JobType]
	jobRegMu.RUnlock()
	if ok && factory != nil {
		j := factory()
		if err := json.Unmarshal(raw, j); err != nil {
			return nil, err
		}
		if err := hydrateEmbeddedBaseJobPayload(raw, j); err != nil {
			return nil, err
		}
		return j, nil
	}
	var b BaseJob
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, err
	}
	if err := hydratePayloadFromFullJSON(raw, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

func embeddedBaseJobPtr(job JobInterface) *BaseJob {
	rv := reflect.ValueOf(job)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return nil
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return nil
	}
	f := rv.FieldByName("BaseJob")
	if !f.IsValid() || f.Kind() != reflect.Struct {
		return nil
	}
	bp, ok := f.Addr().Interface().(*BaseJob)
	if !ok {
		return nil
	}
	return bp
}

func hydrateEmbeddedBaseJobPayload(raw []byte, job JobInterface) error {
	bp := embeddedBaseJobPtr(job)
	if bp == nil {
		return nil
	}
	return hydratePayloadFromFullJSON(raw, bp)
}
