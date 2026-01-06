package services

import (
	"context"
	"fmt"
	"time"

	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"
	"robusta-web/backend/internal/pkg/awx"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// AWXStreamer Handles independent streaming of AWX logs
type AWXStreamer struct {
	db     *gorm.DB
	client *awx.Client
}

func NewAWXStreamer(db *gorm.DB, client *awx.Client) *AWXStreamer {
	return &AWXStreamer{
		db:     db,
		client: client,
	}
}

// StreamJobLogs streams raw stdout logs
func (s *AWXStreamer) StreamJobLogs(ctx context.Context, executionID uint64) (<-chan string, error) {
	outCh := make(chan string, 100)

	go func() {
		defer close(outCh)

		// 1. Find the currently running or most recent AWX Job ID
		jobID, err := s.findActiveAWXJobID(ctx, executionID)
		if err != nil {
			logger.L().Warn("Failed to find active job for log streaming", zap.Uint64("execution_id", executionID), zap.Error(err))
			outCh <- fmt.Sprintf("System: Waiting for AWX Job to start... (%v)\n", err)

			// Retry loop for finding job ID (in case it's in queue)
			retryCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			ticker := time.NewTicker(3 * time.Second)
			defer ticker.Stop()

			found := false
			for {
				select {
				case <-retryCtx.Done():
					return
				case <-ticker.C:
					id, err := s.findActiveAWXJobID(ctx, executionID)
					if err == nil {
						jobID = id
						found = true
						goto FoundJob
					}
				}
			}
		FoundJob:
			if !found {
				return
			}
		}

		logger.L().Info("Starting log stream", zap.Uint64("execution_id", executionID), zap.Int("awx_job_id", jobID))
		outCh <- fmt.Sprintf("System: Connected to AWX Job #%d\n", jobID)

		// 2. Poll for events
		lastEventID := 0
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Fetch events
				// Note: In a real efficient implementation, we should use the `counter` or `id__gt` query param if AWX supports it.
				// For now, we fetch pages. This is a simplified implementation.
				// AWX API /api/v2/jobs/{id}/job_events/?order_by=id&id__gt={last_id}

				// Using the raw client to get filtered events would be better, but let's stick to what we have or extend the client.
				// Since we need to extend the client to fit this efficiently, let's assume we can fetch by page or ID.
				// For this MVP, let's just fetch recent events.
				// Actually, sticking to the standard client `GetJobEvents` which paginates might be slow if we have thousands of logs.
				// Let's optimize: We'll list events sorted by ID descending to check state,
				// but for streaming we need ascending from last known.

				// Let's implementation a custom polling here or use a helper
				events, err := s.fetchNewEvents(ctx, jobID, lastEventID)
				if err != nil {
					logger.L().Error("Failed to fetch events", zap.Error(err))
					continue
				}

				for _, event := range events {
					if event.ID > lastEventID {
						lastEventID = event.ID

						// Determine what to print
						// Standard stdout from AWX
						if event.Stdout != "" {
							outCh <- event.Stdout + "\n"
						}
					}
				}

				// Check if job finished
				job, err := s.client.GetJob(ctx, jobID)
				if err == nil && awx.IsJobFinished(job.Status) {
					outCh <- fmt.Sprintf("\nSystem: Job finished with status: %s\n", job.Status)
					return // End stream
				}
			}
		}
	}()

	return outCh, nil
}

// PipelineNodeStatus represents a visual task node state
type PipelineNodeStatus struct {
	TaskName  string    `json:"task_name"`
	TaskID    string    `json:"task_id"` // uuid or similar
	Status    string    `json:"status"`  // running, success, failed
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	Host      string    `json:"host,omitempty"`
}

// StreamJobProgress streams structured task progress
func (s *AWXStreamer) StreamJobProgress(ctx context.Context, executionID uint64) (<-chan PipelineNodeStatus, error) {
	outCh := make(chan PipelineNodeStatus, 100)

	go func() {
		defer close(outCh)

		jobID, err := s.findActiveAWXJobID(ctx, executionID)
		if err != nil {
			// Retry logic similar to Logs... for brevity, simple wait here
			time.Sleep(5 * time.Second)
			jobID, err = s.findActiveAWXJobID(ctx, executionID)
			if err != nil {
				return
			}
		}

		lastEventID := 0
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		// Cache filtered tasks to avoid duplicate "running" events if not needed,
		// but emitting "running" multiple times is fine for idempotent UI.

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				events, err := s.fetchNewEvents(ctx, jobID, lastEventID)
				if err != nil {
					continue
				}

				for _, event := range events {
					if event.ID > lastEventID {
						lastEventID = event.ID

						// Filter logic
						// We look for: playbook_on_task_start, runner_on_ok, runner_on_failed
						// Skip: runner_on_start (too noisy), playbook_on_vars_prompt

						status := ""
						switch event.Event {
						case "playbook_on_task_start":
							status = "running"
						case "runner_on_ok":
							status = "success"
						case "runner_on_failed", "runner_on_error":
							status = "failed"
						case "runner_on_skipped":
							status = "skipped"
						}

						// Filter out noise
						if status == "" {
							continue
						}

						// Filter out "Gathering Facts" if desired, or keep it.
						// Users usually like seeing "Gathering Facts".

						node := PipelineNodeStatus{
							TaskName:  event.EventData.Task,
							TaskID:    fmt.Sprintf("%d-%s", jobID, event.EventData.Task), // Simple ID
							Status:    status,
							StartTime: event.Created,
							Host:      event.EventData.Host,
						}

						// For completion events, we might want end time
						if status == "success" || status == "failed" {
							node.EndTime = event.Created
						}

						outCh <- node
					}
				}

				// Check finish
				job, err := s.client.GetJob(ctx, jobID)
				if err == nil && awx.IsJobFinished(job.Status) {
					return
				}
			}
		}
	}()

	return outCh, nil
}

// Helpers

func (s *AWXStreamer) findActiveAWXJobID(ctx context.Context, executionID uint64) (int, error) {
	// 1. Get current stage of the execution
	var execution models.PipelineExecution
	if err := s.db.First(&execution, executionID).Error; err != nil {
		return 0, err
	}

	// 2. Find the StageRun for the current stage (or last run stage if finished)
	var stageRun models.StageRun
	if execution.CurrentStageID != "" {
		err := s.db.Where("execution_id = ? AND stage_id = ?", executionID, execution.CurrentStageID).First(&stageRun).Error
		if err == nil && stageRun.AWXJobID != nil {
			return *stageRun.AWXJobID, nil
		}
	}

	// Fallback: Check if there's any running StageRun with AWXJobID
	var runs []models.StageRun
	if err := s.db.Where("execution_id = ? AND awx_job_id IS NOT NULL", executionID).Order("created_at desc").Find(&runs).Error; err != nil {
		return 0, err
	}

	if len(runs) > 0 {
		return *runs[0].AWXJobID, nil
	}

	return 0, fmt.Errorf("no AWX Job ID found for execution %d", executionID)
}

// fetchNewEvents fetches events greater than lastID
// Note: This relies on the client exposing a way to filter by ID, or we fetch generic and filter locally.
// For this implementation, we will assume we need to extend awx.Client or use a raw request here to be efficient.
// To avoid modifying `awx` package too much, let's modify `awx.Client` to support custom query params in `GetJobEvents`.
func (s *AWXStreamer) fetchNewEvents(ctx context.Context, jobID int, lastEventID int) ([]awx.JobEvent, error) {
	// We need to implement a specialized fetcher here that uses `id__gt`
	// Since we defined `s.client` as `*awx.Client`, we should add a method to it or use reflection/public field loop if possible.
	// But `awx.Client` fields are private or we can't easily inject params.
	// The best clean way is to add `GetJobEventsSince(ctx, jobID, lastID)` to `awx.Client`.
	// For now, let's assume `GetJobEvents` returns a page, but that's inefficient.
	// I will update `awx/client.go` to support `id__gt` filtering.

	return s.client.GetJobEventsSince(ctx, jobID, lastEventID)
}
