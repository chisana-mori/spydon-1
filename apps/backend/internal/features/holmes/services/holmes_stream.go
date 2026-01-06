package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"robusta-web/backend/internal/logger"
	"robusta-web/backend/internal/models"

	"go.uber.org/zap"
	"gorm.io/datatypes"
)

func (s *HolmesService) streamInvestigateURL(clusterName string) string {
	// 格式: http://${clusterName}.${holmesGPT.url}/api/stream/investigate
	// 例如: http://kind.holmesgpt.svc:8081/api/stream/investigate
	baseURL := s.config.HolmesGPT.URL
	baseURL = strings.TrimPrefix(baseURL, "http://")
	baseURL = strings.TrimPrefix(baseURL, "https://")
	baseURL = strings.TrimRight(baseURL, "/")
	return fmt.Sprintf("http://%s.%s/api/stream/investigate", clusterName, baseURL)
}

// StartStreamRun 为流式RCA创建运行记录
func (s *HolmesService) StartStreamRun(alertID uint64) (*models.RCARun, error) {
	alert, err := s.getAlertByID(alertID)
	if err != nil {
		return nil, fmt.Errorf("获取告警失败: %w", err)
	}

	run := &models.RCARun{
		AlertID:         alert.ID,
		Status:          string(models.RCAStatusRunning),
		StartedAt:       time.Now(),
		Suspects:        datatypes.JSON([]byte(`{}`)),
		Recommendations: datatypes.JSON([]byte(`{}`)),
		Attachments:     datatypes.JSON([]byte(`{}`)),
	}

	if err := s.db.Create(run).Error; err != nil {
		return nil, fmt.Errorf("创建流式RCA记录失败: %w", err)
	}

	return run, nil
}

// FinalizeStreamRun 在流式RCA结束后更新状态并缓存结果
func (s *HolmesService) FinalizeStreamRun(ctx context.Context, run *models.RCARun, streamChunks []string, metadata map[string]interface{}, streamErr error) {
	if run == nil {
		return
	}

	now := time.Now()

	updates := map[string]interface{}{
		"updated_at": now,
	}

	if streamErr != nil {
		errorMessage := streamErr.Error()
		updates["status"] = string(models.RCAStatusFailed)
		updates["error_message"] = errorMessage
		updates["completed_at"] = now

		run.Status = string(models.RCAStatusFailed)
		run.CompletedAt = &now
		run.ErrorMessage = &errorMessage

		if err := s.db.Model(&models.RCARun{}).Where("id = ?", run.ID).Updates(updates).Error; err != nil {
			logger.L().Error("更新流式RCA失败状态时出错", zap.Error(err), zap.Uint64("rca_run_id", run.ID))
		}
		return
	}

	updates["status"] = string(models.RCAStatusCompleted)
	updates["completed_at"] = now

	// 提取summary和完整文本内容
	var summary string
	var fullText string

	if metadata != nil {
		if s, ok := metadata["summary"].(string); ok {
			summary = s
			updates["summary"] = summary
		}
		if ft, ok := metadata["full_text"].(string); ok {
			fullText = ft
		}
	}

	// 如果metadata中没有full_text，从streamChunks中提取
	if fullText == "" && len(streamChunks) > 0 {
		fullText = s.extractTextFromStreamChunks(streamChunks)
	}

	// 保存到MinIO：将完整的分析结果合并到一个JSON文件
	if s.storage != nil {
		cached := RCACachedResult{
			Version:      "v1",
			RunID:        run.ID,
			AlertID:      run.AlertID,
			CachedAt:     now,
			StreamChunks: streamChunks,
			Metadata: map[string]interface{}{
				"summary":   summary,
				"full_text": fullText,
				"duration":  now.Sub(run.StartedAt).Seconds(),
			},
		}

		payload, err := json.Marshal(cached)
		if err != nil {
			logger.L().Error("序列化RCA流缓存失败", zap.Error(err), zap.Uint64("rca_run_id", run.ID))
		} else {
			prefix := fmt.Sprintf("rca-results/%s", models.FormatID(run.AlertID))
			key, saveErr := s.storage.Save(ctx, prefix, payload, "application/json")
			if saveErr != nil {
				logger.L().Error("保存RCA流缓存失败", zap.Error(saveErr), zap.Uint64("rca_run_id", run.ID))
			} else {
				updates["raw_payload_key"] = key
				run.RawPayloadKey = key
				logger.L().Info("RCA分析结果已缓存到MinIO", zap.Uint64("rca_run_id", run.ID), zap.String("object_key", key), zap.Int("payload_size", len(payload)))
			}
		}
	}

	run.Status = string(models.RCAStatusCompleted)
	run.CompletedAt = &now

	if err := s.db.Model(&models.RCARun{}).Where("id = ?", run.ID).Updates(updates).Error; err != nil {
		logger.L().Error("更新流式RCA运行记录失败", zap.Error(err), zap.Uint64("rca_run_id", run.ID))
	}
}

// Investigate 执行完整的分析流程，包括缓存检查、创建运行记录、流式分析和结果保存
func (s *HolmesService) Investigate(ctx context.Context, alertID uint64, opts InvestigateOptions, streamCallback func(string) error) error {
	if s == nil {
		return fmt.Errorf("HolmesService is not initialized")
	}

	// 1. 尝试从缓存获取
	if opts.PreferCache && !opts.ForceRefresh && s.storage != nil {
		cached, err := s.GetCachedResult(ctx, alertID)
		if err == nil && cached != nil && len(cached.StreamChunks) > 0 {
			// 缓存命中，回放流
			logger.L().Info("RCA缓存命中", zap.Uint64("alert_id", alertID))

			// 发送缓存命中标记（如果需要，可以通过callback发送特定事件，或者由调用方处理header）
			// 这里我们模拟流式回放
			for _, chunk := range cached.StreamChunks {
				if callbackErr := streamCallback(chunk); callbackErr != nil {
					return callbackErr
				}
			}
			return nil
		} else if err != nil {
			logger.L().Warn("读取RCA缓存失败，继续走实时分析", zap.Error(err), zap.Uint64("alert_id", alertID))
		}
	}

	// 2. 创建新的运行记录
	rcaRun, err := s.StartStreamRun(alertID)
	if err != nil {
		logger.L().Warn("创建RCA流式运行记录失败", zap.Error(err), zap.Uint64("alert_id", alertID))
		// 即使创建记录失败，也可以尝试继续分析，只是无法保存结果
	}

	// 3. 获取告警信息
	alert, err := s.getAlertByID(alertID)
	if err != nil {
		return fmt.Errorf("获取告警失败: %w", err)
	}

	// 3.5 如果未显式提供knowledge base，则尝试按规则名加载一份提示词
	if opts.KnowledgeBase == "" && s.knowledge != nil && alert != nil {
		if kb, kbErr := s.knowledge.BuildKnowledgeBaseForRule(ctx, alert.Title); kbErr == nil && kb != "" {
			opts.KnowledgeBase = kb
		} else if kbErr != nil {
			logger.L().Warn("加载经验指南作为提示词失败", zap.Error(kbErr), zap.Uint64("alert_id", alertID))
		}
	}

	// 4. 执行流式分析
	var streamErr error
	var streamChunks []string
	metadata := make(map[string]interface{})

	defer func() {
		// 5. 结束分析，保存结果
		if rcaRun != nil {
			logger.L().Info("流式RCA分析完成，准备保存", zap.Int("chunk_count", len(streamChunks)), zap.Error(streamErr))
			s.FinalizeStreamRun(ctx, rcaRun, streamChunks, metadata, streamErr)
		}
	}()

	streamChunks, err = s.StreamInvestigateChunks(ctx, alert, opts, func(chunk string) error {
		// 转发给调用方
		if callbackErr := streamCallback(chunk); callbackErr != nil {
			return callbackErr
		}
		return nil
	})

	if err != nil {
		streamErr = err
		return err
	}

	return nil
}

// StreamInvestigateChunks 调用HolmesGPT的流式分析接口
func (s *HolmesService) StreamInvestigateChunks(
	ctx context.Context,
	alert *models.Alert,
	opts InvestigateOptions,
	consumer func(string) error,
) ([]string, error) {
	if s == nil {
		return nil, fmt.Errorf("HolmesService is not initialized")
	}
	if alert == nil {
		return nil, fmt.Errorf("alert is nil")
	}

	payload, err := s.buildInvestigatePayload(alert, opts)
	if err != nil {
		return nil, err
	}
	resp, err := s.streamClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "text/event-stream").
		SetHeader("Cache-Control", "no-cache").
		SetBody(bytes.NewReader(payload)).
		SetDoNotParseResponse(true).
		Post(s.streamInvestigateURL(alert.ClusterName))
	if err != nil {
		return nil, fmt.Errorf("HolmesGPT流式请求失败: %w", err)
	}
	raw := resp.RawBody()
	defer func() { _ = raw.Close() }()

	if resp.StatusCode() != http.StatusOK {
		body, _ := io.ReadAll(raw)
		return nil, fmt.Errorf("HolmesGPT流式分析返回 %d: %s", resp.StatusCode(), string(body))
	}

	return s.processStreamResponse(ctx, raw, consumer)
}

// processStreamResponse 处理流式响应
func (s *HolmesService) processStreamResponse(ctx context.Context, raw io.Reader, consumer func(string) error) ([]string, error) {
	chunks := make([]string, 0, 128)
	buffer := make([]byte, 4096)
	buffered := ""

	for {
		if err := ctx.Err(); err != nil {
			return chunks, err
		}
		n, readErr := raw.Read(buffer)
		if n > 0 {
			buffered += string(buffer[:n])
			for {
				idx := strings.Index(buffered, "\n\n")
				if idx < 0 {
					break
				}
				event := buffered[:idx+2]
				chunks = append(chunks, event)
				if consumer != nil {
					if err := consumer(event); err != nil {
						return chunks, err
					}
				}
				buffered = buffered[idx+2:]
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				if strings.TrimSpace(buffered) != "" {
					event := buffered
					chunks = append(chunks, event)
					if consumer != nil {
						if err := consumer(event); err != nil {
							return chunks, err
						}
					}
				}
				return chunks, nil
			}
			return chunks, readErr
		}
	}
}

// extractTextFromStreamChunks 从SSE流chunks中提取纯文本内容
func (s *HolmesService) extractTextFromStreamChunks(chunks []string) string {
	var textParts []string

	for _, chunk := range chunks {
		// 解析SSE格式: data: {...}
		lines := strings.Split(chunk, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			dataStr := strings.TrimPrefix(line, "data: ")
			if dataStr == "[DONE]" || dataStr == "" {
				continue
			}

			// 尝试解析JSON
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(dataStr), &data); err != nil {
				continue
			}

			// 提取content字段
			if content, ok := data["content"].(string); ok && content != "" {
				textParts = append(textParts, content)
			}
		}
	}

	return strings.Join(textParts, "")
}
