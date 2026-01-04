package awx

import "time"

// JobTemplate AWX Job Template
type JobTemplate struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description"`
	JobType          string `json:"job_type"`
	Inventory        int    `json:"inventory"`
	Project          int    `json:"project"`
	Playbook         string `json:"playbook"`
	AskVarsOnLaunch  bool   `json:"ask_variables_on_launch"`
	AskLimitOnLaunch bool   `json:"ask_limit_on_launch"`
	ExtraVars        string `json:"extra_vars"` // usually YAML/JSON string
}

// Job AWX Job 实例
type Job struct {
	ID              int        `json:"id"`
	Name            string     `json:"name"`
	Status          string     `json:"status"` // pending, waiting, running, successful, failed, error, canceled
	Failed          bool       `json:"failed"`
	Started         *time.Time `json:"started"`
	Finished        *time.Time `json:"finished"`
	Elapsed         float64    `json:"elapsed"`
	JobType         string     `json:"job_type"`
	LaunchType      string     `json:"launch_type"`
	ResultTraceback string     `json:"result_traceback"`
	JobExplanation  string     `json:"job_explanation"`
}

// JobLaunchRequest 启动Job的请求参数
type JobLaunchRequest struct {
	ExtraVars map[string]interface{} `json:"extra_vars,omitempty"`
	Limit     string                 `json:"limit,omitempty"`
	JobType   string                 `json:"job_type,omitempty"` // run or check (dry-run)
	Diff      bool                   `json:"diff,omitempty"`     // 显示变更差异
}

// JobLaunchResponse 启动Job后的响应
type JobLaunchResponse struct {
	Job           int                    `json:"job"`
	IgnoredFields map[string]interface{} `json:"ignored_fields,omitempty"`
	ID            int                    `json:"id"`
	Type          string                 `json:"type"`
	URL           string                 `json:"url"`
	CreatedBy     int                    `json:"created_by"`
	Status        string                 `json:"status"`
}

// JobEvent AWX Job 事件 (用于日志)
type JobEvent struct {
	ID         int       `json:"id"`
	Counter    int       `json:"counter"`
	EventLevel int       `json:"event_level"`
	Event      string    `json:"event"` // playbook_on_start, runner_on_ok, runner_on_failed etc.
	EventData  EventData `json:"event_data"`
	Stdout     string    `json:"stdout"`
	Created    time.Time `json:"created"`
	Changed    bool      `json:"changed"`
	Failed     bool      `json:"failed"`
}

// EventData Job事件的详细数据
type EventData struct {
	Playbook string      `json:"playbook"`
	Play     string      `json:"play"`
	Task     string      `json:"task"`
	Host     string      `json:"host"`
	Role     string      `json:"role"`
	Res      interface{} `json:"res"` // Ansible result
}

// PaginatedResponse AWX分页响应的通用结构
type PaginatedResponse[T any] struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []T    `json:"results"`
}

// Inventory AWX Inventory
type Inventory struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	TotalHosts  int    `json:"total_hosts"`
	TotalGroups int    `json:"total_groups"`
}

// JobStatus Job 状态常量
const (
	JobStatusPending    = "pending"
	JobStatusWaiting    = "waiting"
	JobStatusRunning    = "running"
	JobStatusSuccessful = "successful"
	JobStatusFailed     = "failed"
	JobStatusError      = "error"
	JobStatusCanceled   = "canceled"
)

// IsJobFinished 判断Job是否已完成
func IsJobFinished(status string) bool {
	switch status {
	case JobStatusSuccessful, JobStatusFailed, JobStatusError, JobStatusCanceled:
		return true
	default:
		return false
	}
}

// IsJobSuccessful 判断Job是否成功
func IsJobSuccessful(status string) bool {
	return status == JobStatusSuccessful
}
