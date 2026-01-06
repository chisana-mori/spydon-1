package http

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	sharedservices "robusta-web/backend/internal/features/shared/services"
	"robusta-web/backend/internal/logger"
)

// EnrichmentProcessor 处理enrichments的处理器
type EnrichmentProcessor struct {
	storageService sharedservices.PayloadStorage
}

// NewEnrichmentProcessor 创建enrichment处理器
func NewEnrichmentProcessor(storageService sharedservices.PayloadStorage) *EnrichmentProcessor {
	return &EnrichmentProcessor{
		storageService: storageService,
	}
}

// ProcessEnrichments 处理enrichments并保存到存储
func (p *EnrichmentProcessor) ProcessEnrichments(ctx context.Context, clusterID, fingerprint string, enrichments []RobustaEnrichment) map[string]string {
	if p.storageService == nil || len(enrichments) == 0 {
		return nil
	}

	enrichmentKeys := make(map[string]string)
	basePath := fmt.Sprintf("enrichments/%s/%s", clusterID, fingerprint)

	for i, enr := range enrichments {
		enrichmentID := p.buildEnrichmentID(enr, i)
		p.saveEnrichmentMetadata(ctx, basePath, enrichmentID, enr, enrichmentKeys)
		p.processBlocks(ctx, basePath, enrichmentID, enr.Blocks, enrichmentKeys)
	}

	return enrichmentKeys
}

// buildEnrichmentID 构建enrichment ID
func (p *EnrichmentProcessor) buildEnrichmentID(enr RobustaEnrichment, index int) string {
	return fmt.Sprintf("%s_%d", enr.Title, index)
}

// saveEnrichmentMetadata 保存enrichment元数据
func (p *EnrichmentProcessor) saveEnrichmentMetadata(ctx context.Context, basePath, enrichmentID string, enr RobustaEnrichment, enrichmentKeys map[string]string) {
	enrichmentJSON, err := json.MarshalIndent(enr, "", "  ")
	if err != nil {
		return
	}

	enrichmentPath := fmt.Sprintf("%s/%s.json", basePath, enrichmentID)
	if key, err := p.storageService.Save(ctx, enrichmentPath, enrichmentJSON, "application/json"); err == nil {
		enrichmentKeys[enrichmentID] = key
	}
}

// processBlocks 处理所有blocks
func (p *EnrichmentProcessor) processBlocks(ctx context.Context, basePath, enrichmentID string, blocks []RobustaBlock, enrichmentKeys map[string]string) {
	for i, block := range blocks {
		blockKey := fmt.Sprintf("%s_block_%d", enrichmentID, i)
		p.processBlock(ctx, basePath, enrichmentID, i, block, blockKey, enrichmentKeys)
	}
}

// processBlock 处理单个block
func (p *EnrichmentProcessor) processBlock(ctx context.Context, basePath, enrichmentID string, index int, block RobustaBlock, blockKey string, enrichmentKeys map[string]string) {
	switch block.Type {
	case "FileBlock", "GraphBlock":
		p.processFileBlock(ctx, basePath, enrichmentID, index, block, blockKey, enrichmentKeys)
	case "MarkdownBlock":
		p.processMarkdownBlock(ctx, basePath, enrichmentID, index, block, blockKey, enrichmentKeys)
	case "TableBlock":
		p.processTableBlock(ctx, basePath, enrichmentID, index, block, blockKey, enrichmentKeys)
	case "JsonBlock":
		p.processJsonBlock(ctx, basePath, enrichmentID, index, block, blockKey, enrichmentKeys)
	case "ListBlock":
		p.processListBlock(ctx, basePath, enrichmentID, index, block, blockKey, enrichmentKeys)
	case "KubernetesDiffBlock":
		p.processDiffBlock(ctx, basePath, enrichmentID, index, block, blockKey, enrichmentKeys)
	case "EventsBlock":
		p.processEventsBlock(ctx, basePath, enrichmentID, index, block, blockKey, enrichmentKeys)
	case "PrometheusBlock":
		p.processPrometheusBlock(ctx, basePath, enrichmentID, index, block, blockKey, enrichmentKeys)
	case "LinksBlock":
		p.processLinksBlock(ctx, basePath, enrichmentID, index, block, blockKey, enrichmentKeys)
	case "CallbackBlock":
		p.processCallbackBlock(ctx, basePath, enrichmentID, index, block, blockKey, enrichmentKeys)
	case "HeaderBlock":
		logger.S().Infow("处理HeaderBlock", "title", block.Title)
	default:
		p.processUnknownBlock(ctx, basePath, enrichmentID, index, block, blockKey, enrichmentKeys)
	}
}

// processFileBlock 处理文件/图表内容
func (p *EnrichmentProcessor) processFileBlock(ctx context.Context, basePath, enrichmentID string, index int, block RobustaBlock, blockKey string, enrichmentKeys map[string]string) {
	if block.Contents == "" {
		return
	}

	decoded, err := decodeBase64Content(block.Contents)
	if err != nil {
		logger.S().Errorw("解码块内容失败", "block_type", block.Type, "error", err)
		return
	}

	filename := block.Filename
	if filename == "" {
		if block.Type == "GraphBlock" {
			filename = fmt.Sprintf("graph_%d.png", index)
		} else {
			filename = fmt.Sprintf("file_%d", index)
		}
	}

	contentType := guessContentType(filename, decoded)
	filePath := fmt.Sprintf("%s/%s_files/%s", basePath, enrichmentID, filename)

	if key, err := p.storageService.Save(ctx, filePath, decoded, contentType); err == nil {
		enrichmentKeys[blockKey] = key
		logger.S().Infow("块内容已保存", "block_type", block.Type, "filename", filename, "object_key", key)
	}
}

// processMarkdownBlock 处理Markdown内容
func (p *EnrichmentProcessor) processMarkdownBlock(ctx context.Context, basePath, enrichmentID string, index int, block RobustaBlock, blockKey string, enrichmentKeys map[string]string) {
	if block.Text == "" {
		return
	}

	filePath := fmt.Sprintf("%s/%s_markdown_%d.md", basePath, enrichmentID, index)
	if key, err := p.storageService.Save(ctx, filePath, []byte(block.Text), "text/markdown"); err == nil {
		enrichmentKeys[blockKey] = key
	}
}

// processTableBlock 处理表格数据
func (p *EnrichmentProcessor) processTableBlock(ctx context.Context, basePath, enrichmentID string, index int, block RobustaBlock, blockKey string, enrichmentKeys map[string]string) {
	if len(block.Headers) == 0 && len(block.Rows) == 0 {
		return
	}

	tableData := map[string]interface{}{
		"headers": block.Headers,
		"rows":    block.Rows,
	}
	tableJSON, _ := json.MarshalIndent(tableData, "", "  ")
	filePath := fmt.Sprintf("%s/%s_table_%d.json", basePath, enrichmentID, index)

	if key, err := p.storageService.Save(ctx, filePath, tableJSON, "application/json"); err == nil {
		enrichmentKeys[blockKey] = key
	}
}

// processJsonBlock 处理JSON数据
func (p *EnrichmentProcessor) processJsonBlock(ctx context.Context, basePath, enrichmentID string, index int, block RobustaBlock, blockKey string, enrichmentKeys map[string]string) {
	if block.JSON == nil {
		return
	}

	jsonData, _ := json.MarshalIndent(block.JSON, "", "  ")
	filePath := fmt.Sprintf("%s/%s_json_%d.json", basePath, enrichmentID, index)

	if key, err := p.storageService.Save(ctx, filePath, jsonData, "application/json"); err == nil {
		enrichmentKeys[blockKey] = key
	}
}

// processListBlock 处理列表数据
func (p *EnrichmentProcessor) processListBlock(ctx context.Context, basePath, enrichmentID string, index int, block RobustaBlock, blockKey string, enrichmentKeys map[string]string) {
	if len(block.Items) == 0 {
		return
	}

	listData := map[string]interface{}{
		"items": block.Items,
	}
	listJSON, _ := json.MarshalIndent(listData, "", "  ")
	filePath := fmt.Sprintf("%s/%s_list_%d.json", basePath, enrichmentID, index)

	if key, err := p.storageService.Save(ctx, filePath, listJSON, "application/json"); err == nil {
		enrichmentKeys[blockKey] = key
	}
}

// processDiffBlock 处理Kubernetes差异
func (p *EnrichmentProcessor) processDiffBlock(ctx context.Context, basePath, enrichmentID string, index int, block RobustaBlock, blockKey string, enrichmentKeys map[string]string) {
	if len(block.Diffs) == 0 && block.Old == "" && block.New == "" {
		return
	}

	diffData := map[string]interface{}{
		"diffs": block.Diffs,
		"old":   block.Old,
		"new":   block.New,
	}
	diffJSON, _ := json.MarshalIndent(diffData, "", "  ")
	filePath := fmt.Sprintf("%s/%s_diff_%d.json", basePath, enrichmentID, index)

	if key, err := p.storageService.Save(ctx, filePath, diffJSON, "application/json"); err == nil {
		enrichmentKeys[blockKey] = key
	}
}

// processEventsBlock 处理事件数据
func (p *EnrichmentProcessor) processEventsBlock(ctx context.Context, basePath, enrichmentID string, index int, block RobustaBlock, blockKey string, enrichmentKeys map[string]string) {
	if len(block.Events) == 0 {
		return
	}

	eventsData := map[string]interface{}{
		"events": block.Events,
	}
	eventsJSON, _ := json.MarshalIndent(eventsData, "", "  ")
	filePath := fmt.Sprintf("%s/%s_events_%d.json", basePath, enrichmentID, index)

	if key, err := p.storageService.Save(ctx, filePath, eventsJSON, "application/json"); err == nil {
		enrichmentKeys[blockKey] = key
		logger.S().Infow("EventsBlock已保存", "count", len(block.Events), "object_key", key)
	}
}

// processPrometheusBlock 处理Prometheus查询结果
func (p *EnrichmentProcessor) processPrometheusBlock(ctx context.Context, basePath, enrichmentID string, index int, block RobustaBlock, blockKey string, enrichmentKeys map[string]string) {
	var data []byte

	if len(block.Headers) > 0 || len(block.Rows) > 0 {
		tbl := map[string]interface{}{"headers": block.Headers, "rows": block.Rows}
		data, _ = json.MarshalIndent(tbl, "", "  ")
	} else if block.JSON != nil {
		data, _ = json.MarshalIndent(block.JSON, "", "  ")
	} else if block.Metadata != nil {
		data, _ = json.MarshalIndent(block.Metadata, "", "  ")
	}

	if len(data) == 0 {
		return
	}

	filePath := fmt.Sprintf("%s/%s_prometheus_%d.json", basePath, enrichmentID, index)
	if key, err := p.storageService.Save(ctx, filePath, data, "application/json"); err == nil {
		enrichmentKeys[blockKey] = key
		logger.S().Infow("PrometheusBlock已保存", "index", index, "object_key", key)
	}
}

// processLinksBlock 处理链接集合
func (p *EnrichmentProcessor) processLinksBlock(ctx context.Context, basePath, enrichmentID string, index int, block RobustaBlock, blockKey string, enrichmentKeys map[string]string) {
	if block.Metadata == nil {
		return
	}

	linksJSON, _ := json.MarshalIndent(block.Metadata, "", "  ")
	filePath := fmt.Sprintf("%s/%s_links_%d.json", basePath, enrichmentID, index)

	if key, err := p.storageService.Save(ctx, filePath, linksJSON, "application/json"); err == nil {
		enrichmentKeys[blockKey] = key
		logger.S().Infow("LinksBlock已保存", "index", index, "object_key", key)
	}
}

// processCallbackBlock 处理交互按钮
func (p *EnrichmentProcessor) processCallbackBlock(ctx context.Context, basePath, enrichmentID string, index int, block RobustaBlock, blockKey string, enrichmentKeys map[string]string) {
	if block.Metadata == nil {
		return
	}

	cbJSON, _ := json.MarshalIndent(block.Metadata, "", "  ")
	filePath := fmt.Sprintf("%s/%s_callback_%d.json", basePath, enrichmentID, index)

	if key, err := p.storageService.Save(ctx, filePath, cbJSON, "application/json"); err == nil {
		enrichmentKeys[blockKey] = key
		logger.S().Infow("CallbackBlock已保存", "index", index, "object_key", key)
	}
}

// processUnknownBlock 处理未知类型的block
func (p *EnrichmentProcessor) processUnknownBlock(ctx context.Context, basePath, enrichmentID string, index int, block RobustaBlock, blockKey string, enrichmentKeys map[string]string) {
	unknownJSON, _ := json.MarshalIndent(block, "", "  ")
	filePath := fmt.Sprintf("%s/%s_unknown_%d.json", basePath, enrichmentID, index)

	if key, err := p.storageService.Save(ctx, filePath, unknownJSON, "application/json"); err == nil {
		enrichmentKeys[blockKey] = key
		logger.S().Infow("未知Block已保存", "block_type", block.Type, "index", index, "object_key", key)
	} else {
		logger.S().Errorw("保存未知Block失败", "block_type", block.Type, "index", index, "error", err)
	}
}

// decodeBase64Content 解码base64/hex内容，失败时按原文返回
func decodeBase64Content(content string) ([]byte, error) {
	// 优先尝试标准 base64
	if b, err := base64.StdEncoding.DecodeString(content); err == nil {
		return b, nil
	}
	// 尝试无填充 base64
	if b, err := base64.RawStdEncoding.DecodeString(content); err == nil {
		return b, nil
	}
	// 尝试 URL base64
	if b, err := base64.URLEncoding.DecodeString(content); err == nil {
		return b, nil
	}
	// 回退尝试十六进制
	if b, err := hex.DecodeString(content); err == nil {
		return b, nil
	}
	// 最后回退
	return []byte(content), nil
}

// guessContentType 根据文件名和内容猜测内容类型
func guessContentType(filename string, content []byte) string {
	// 根据文件扩展名
	if strings.HasSuffix(filename, ".json") {
		return "application/json"
	}
	if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		return "application/yaml"
	}
	if strings.HasSuffix(filename, ".txt") || strings.HasSuffix(filename, ".log") {
		return "text/plain"
	}
	if strings.HasSuffix(filename, ".md") || strings.HasSuffix(filename, ".markdown") {
		return "text/markdown"
	}
	if strings.HasSuffix(filename, ".csv") {
		return "text/csv"
	}
	if strings.HasSuffix(filename, ".html") {
		return "text/html"
	}
	if strings.HasSuffix(filename, ".xml") {
		return "application/xml"
	}
	if strings.HasSuffix(filename, ".png") {
		return "image/png"
	}
	if strings.HasSuffix(filename, ".jpg") || strings.HasSuffix(filename, ".jpeg") {
		return "image/jpeg"
	}
	if strings.HasSuffix(filename, ".gif") {
		return "image/gif"
	}
	if strings.HasSuffix(filename, ".svg") {
		return "image/svg+xml"
	}

	// 默认
	return "application/octet-stream"
}
