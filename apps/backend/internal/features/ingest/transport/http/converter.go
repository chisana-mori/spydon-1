package http

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"robusta-web/backend/internal/models"
	"robusta-web/backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/datatypes"
)

// FindingToAlertConverter Finding到Alert的转换器
type FindingToAlertConverter struct {
	ctx            context.Context
	ginCtx         *gin.Context
	handler        *Handler
	finding        *RobustaFinding
	rawPayload     []byte
	enrichmentProc *EnrichmentProcessor
	clusterName    string
	title          string
	description    string
	severity       string
	fingerprint    string
	aggregationKey string
	startsAt       *time.Time
	endsAt         *time.Time
	labels         map[string]any
	annotations    map[string]any
}

// NewFindingToAlertConverter 创建新的转换器（使用gin.Context）
func NewFindingToAlertConverter(ginCtx *gin.Context, handler *Handler, finding *RobustaFinding, rawPayload []byte) *FindingToAlertConverter {
	return &FindingToAlertConverter{
		ctx:            ginCtx.Request.Context(),
		ginCtx:         ginCtx,
		handler:        handler,
		finding:        finding,
		rawPayload:     rawPayload,
		enrichmentProc: NewEnrichmentProcessor(handler.storageService),
		labels:         make(map[string]any),
		annotations:    make(map[string]any),
	}
}

// NewFindingToAlertConverterWithContext 创建新的转换器（使用独立context）
func NewFindingToAlertConverterWithContext(ctx context.Context, handler *Handler, finding *RobustaFinding, rawPayload []byte) *FindingToAlertConverter {
	return &FindingToAlertConverter{
		ctx:            ctx,
		ginCtx:         nil,
		handler:        handler,
		finding:        finding,
		rawPayload:     rawPayload,
		enrichmentProc: NewEnrichmentProcessor(handler.storageService),
		labels:         make(map[string]any),
		annotations:    make(map[string]any),
	}
}

// Convert 执行转换
func (c *FindingToAlertConverter) Convert() (*models.Alert, error) {
	c.extractClusterName()
	c.buildTitle()
	c.buildDescription()
	c.normalizeSeverity()
	c.buildAggregationKey()
	c.parseTimes()
	c.buildLabels()
	c.buildAnnotations()

	payloadKey, err := c.saveRawPayload()
	if err != nil {
		logger.L().Error("保存原始告警数据失败", zap.Error(err))
	}

	enrichmentKeys := c.processEnrichments()
	if len(enrichmentKeys) > 0 {
		enrichKeysJSON, _ := json.Marshal(enrichmentKeys)
		c.annotations["enrichment_keys"] = string(enrichKeysJSON)
	}

	labelsJSON, annotationsJSON := c.serializeMetadata()
	status := c.determineStatus()

	alert := &models.Alert{
		Fingerprint:   c.fingerprint,
		ClusterName:   c.clusterName,
		Title:         c.title,
		Description:   c.description,
		Severity:      c.severity,
		Status:        status,
		Labels:        labelsJSON,
		Annotations:   annotationsJSON,
		StartsAt:      c.startsAt,
		EndsAt:        c.endsAt,
		RawPayloadKey: payloadKey,
	}

	return alert, nil
}

// extractClusterName 提取cluster_name
func (c *FindingToAlertConverter) extractClusterName() {
	c.clusterName = extractClusterName(c.ginCtx, c.finding)
}

// buildTitle 构建标题
func (c *FindingToAlertConverter) buildTitle() {
	c.title = c.finding.Title

	if c.title == "" {
		subjectName := c.finding.Subject.Name
		subjectNamespace := c.finding.Subject.Namespace
		subjectType := c.finding.Subject.SubjectType.String()

		if subjectName != "" {
			c.title = fmt.Sprintf("%s: %s", subjectType, subjectName)
			if subjectNamespace != "" {
				c.title = fmt.Sprintf("%s (%s)", c.title, subjectNamespace)
			}
		}
	}

	if c.title == "" {
		c.title = "Robusta Finding"
	}
}

// buildDescription 构建描述
func (c *FindingToAlertConverter) buildDescription() {
	c.description = c.finding.Description
}

// normalizeSeverity 标准化严重级别
func (c *FindingToAlertConverter) normalizeSeverity() {
	severityValue := fmt.Sprintf("%d", c.finding.Severity.Value)
	c.severity = normalizeSeverity(severityValue)
}

// buildAggregationKey 构建聚合键
func (c *FindingToAlertConverter) buildAggregationKey() {
	c.aggregationKey = c.finding.AggregationKey

	if c.aggregationKey == "" {
		subjectNamespace := c.finding.Subject.Namespace
		subjectName := c.finding.Subject.Name
		if subjectNamespace != "" && subjectName != "" {
			c.aggregationKey = fmt.Sprintf("%s/%s", subjectNamespace, subjectName)
		}
	}
}

// parseTimes 解析时间
func (c *FindingToAlertConverter) parseTimes() {
	if !c.finding.StartsAt.IsZero() {
		c.startsAt = &c.finding.StartsAt.Time
	}
	if c.finding.EndsAt != nil && !c.finding.EndsAt.IsZero() {
		c.endsAt = &c.finding.EndsAt.Time
	}
}

// buildLabels 构建labels
func (c *FindingToAlertConverter) buildLabels() {
	if c.finding.Subject.Labels != nil {
		for k, v := range c.finding.Subject.Labels {
			c.labels[k] = v
		}
	}

	if c.finding.SilenceLabels != nil {
		for k, v := range c.finding.SilenceLabels {
			c.labels[k] = v
		}
	}

	c.labels["source"] = c.finding.Source.Value
	c.labels["finding_type"] = c.finding.FindingType.Value

	c.labels["subject_type"] = c.finding.Subject.SubjectType.String()
	if c.finding.Subject.Namespace != "" {
		c.labels["namespace"] = c.finding.Subject.Namespace
	}
	if c.finding.Subject.Node != "" {
		c.labels["node"] = c.finding.Subject.Node
	}
}

// buildAnnotations 构建annotations
func (c *FindingToAlertConverter) buildAnnotations() {
	if c.finding.Subject.Annotations != nil {
		for k, v := range c.finding.Subject.Annotations {
			c.annotations[k] = v
		}
	}

	if c.finding.InvestigateURI != "" {
		c.annotations["investigate_uri"] = c.finding.InvestigateURI
	}
	if c.finding.Service != nil {
		serviceJSON, _ := json.Marshal(c.finding.Service)
		c.annotations["service"] = string(serviceJSON)
	}
	if c.finding.ServiceKey != "" {
		c.annotations["service_key"] = c.finding.ServiceKey
	}

	if len(c.finding.Links) > 0 {
		linksJSON, _ := json.Marshal(c.finding.Links)
		c.annotations["links"] = string(linksJSON)
	}
}

// saveRawPayload 保存原始payload
func (c *FindingToAlertConverter) saveRawPayload() (string, error) {
	if c.handler.storageService == nil {
		return "", nil
	}

	key, err := c.handler.storageService.Save(
		c.ctx,
		fmt.Sprintf("alerts/%s", c.clusterName),
		c.rawPayload,
		"application/json",
	)
	if err == nil {
		logger.L().Debug("原始告警数据已保存", zap.String("object_key", key))
	}
	return key, err
}

// processEnrichments 处理enrichments
func (c *FindingToAlertConverter) processEnrichments() map[string]string {
	return c.enrichmentProc.ProcessEnrichments(
		c.ctx,
		c.clusterName,
		c.fingerprint,
		c.finding.Enrichments,
	)
}

// serializeMetadata 序列化labels和annotations
func (c *FindingToAlertConverter) serializeMetadata() (datatypes.JSON, datatypes.JSON) {
	var labelsJSON, annotationsJSON datatypes.JSON

	if len(c.labels) > 0 {
		if b, err := json.Marshal(c.labels); err == nil {
			labelsJSON = datatypes.JSON(b)
		}
	}

	if len(c.annotations) > 0 {
		if b, err := json.Marshal(c.annotations); err == nil {
			annotationsJSON = datatypes.JSON(b)
		}
	}

	return labelsJSON, annotationsJSON
}

// determineStatus 确定状态
func (c *FindingToAlertConverter) determineStatus() string {
	if c.endsAt != nil && c.endsAt.Before(time.Now()) {
		return "resolved"
	}
	return "firing"
}
