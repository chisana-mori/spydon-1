package services

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"robusta-web/backend/internal/db"
	sharedservices "robusta-web/backend/internal/features/shared/services"
	"robusta-web/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// KnowledgeService 知识库服务
type KnowledgeService struct {
	db      *db.Database
	storage sharedservices.PayloadStorage
}

func NewKnowledgeService(database *db.Database, storage sharedservices.PayloadStorage) *KnowledgeService {
	return &KnowledgeService{db: database, storage: storage}
}

// Normalize 规范化规则名：去空白、统一分隔符为'-'、小写
var nonWord = regexp.MustCompile(`[^\p{Han}\w]+`) // 保留中文、字母数字、下划线

func (s *KnowledgeService) Normalize(name string) string {
	n := strings.TrimSpace(name)
	n = strings.ReplaceAll(n, "_", "-")
	n = nonWord.ReplaceAllString(n, "-")
	n = strings.ToLower(n)
	n = strings.Trim(n, "-")
	return n
}

// Manifest 结构（与设计文档一致的最小子集）
type Manifest struct {
	Schema    string                   `json:"schema"`
	ArticleID string                   `json:"articleId"`
	AlertRule string                   `json:"alertRuleName"`
	Tags      []string                 `json:"tags,omitempty"`
	Status    string                   `json:"status"`
	Version   int                      `json:"version"`
	Excerpt   string                   `json:"excerpt,omitempty"`
	Content   map[string]interface{}   `json:"content"`
	Markdown  string                   `json:"markdown,omitempty"`
	Assets    []map[string]interface{} `json:"assets,omitempty"`
	Scopes    map[string]interface{}   `json:"scopes,omitempty"`
	Audit     map[string]interface{}   `json:"audit,omitempty"`
}

func (s *KnowledgeService) manifestKey(articleID string) string {
	return path.Join("knowledge", articleID, "manifest.json")
}

func (s *KnowledgeService) versionKey(articleID string, version int) string {
	return path.Join("knowledge", articleID, "versions", fmt.Sprintf("%d", version), "manifest.json")
}

// Create 创建知识条目，初始化manifest
func (s *KnowledgeService) Create(ctx context.Context, alertRuleName string, tags []string, user string) (*models.KnowledgeArticle, error) {
	articleID := fmt.Sprintf("%d", time.Now().UnixNano())
	objectKey := s.manifestKey(articleID)

	norm := s.Normalize(alertRuleName)
	tagsJSON, _ := json.Marshal(tags)

	art := &models.KnowledgeArticle{
		AlertRuleName:           alertRuleName,
		AlertRuleNameNormalized: norm,
		Tags:                    datatypes.JSON(tagsJSON),
		Status:                  "draft",
		ObjectKey:               objectKey,
		Version:                 1,
		CreatedBy:               user,
		UpdatedBy:               user,
	}

	if err := s.db.DB.Create(art).Error; err != nil {
		return nil, err
	}

	// 初始化manifest
	man := Manifest{
		Schema:    "kb-manifest@v1",
		ArticleID: articleID,
		AlertRule: alertRuleName,
		Tags:      tags,
		Status:    art.Status,
		Version:   art.Version,
		Content:   map[string]interface{}{"tiptap": map[string]interface{}{"type": "doc", "content": []interface{}{}}},
		Markdown:  "",
		Audit: map[string]interface{}{
			"createdBy": user,
			"createdAt": time.Now().UTC().Format(time.RFC3339),
			"updatedBy": user,
			"updatedAt": time.Now().UTC().Format(time.RFC3339),
		},
	}
	b, _ := json.Marshal(man)
	if err := s.storage.PutAt(ctx, objectKey, b, "application/json"); err != nil {
		return nil, fmt.Errorf("写入manifest失败: %w", err)
	}

	return art, nil
}

// Update 更新manifest与元数据（不改变status）
func (s *KnowledgeService) Update(ctx context.Context, id uint64, payload Manifest, user string) (*models.KnowledgeArticle, error) {
	var art models.KnowledgeArticle
	if err := s.db.DB.First(&art, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// 合并关键信息
	if len(payload.Tags) > 0 {
		if b, err := json.Marshal(payload.Tags); err == nil {
			art.Tags = datatypes.JSON(b)
		}
	}
	art.UpdatedBy = user
	art.Version = art.Version + 1

	if err := s.db.DB.Save(&art).Error; err != nil {
		return nil, err
	}

	// 写manifest（保持 objectKey 不变）
	payload.Schema = "kb-manifest@v1"
	payload.ArticleID = strings.TrimPrefix(art.ObjectKey, "knowledge/")
	payload.Status = art.Status
	payload.Version = art.Version
	if payload.Audit == nil {
		payload.Audit = map[string]interface{}{}
	}
	payload.Audit["updatedBy"] = user
	payload.Audit["updatedAt"] = time.Now().UTC().Format(time.RFC3339)

	b, _ := json.Marshal(payload)
	if err := s.storage.PutAt(ctx, art.ObjectKey, b, "application/json"); err != nil {
		return nil, fmt.Errorf("写入manifest失败: %w", err)
	}
	return &art, nil
}

// Publish 发布（复制当前manifest至 versions/{version} 并置为 published）
func (s *KnowledgeService) Publish(ctx context.Context, id uint64, changeSummary, user string) error {
	var art models.KnowledgeArticle
	if err := s.db.DB.First(&art, "id = ?", id).Error; err != nil {
		return err
	}
	// 读取当前manifest
	data, err := s.storage.Get(ctx, art.ObjectKey)
	if err != nil {
		return err
	}

	// 保存版本副本
	vkey := s.versionKey(strings.TrimPrefix(art.ObjectKey, "knowledge/"), art.Version)
	if err := s.storage.PutAt(ctx, vkey, data, "application/json"); err != nil {
		return fmt.Errorf("保存版本副本失败: %w", err)
	}

	// 写版本记录
	ver := &models.KnowledgeArticleVersion{
		ArticleID:     art.ID,
		Version:       art.Version,
		ObjectKey:     vkey,
		ChangeSummary: changeSummary,
		CreatedBy:     user,
		CreatedAt:     time.Now(),
	}
	if err := s.db.DB.Create(ver).Error; err != nil {
		return err
	}

	// 更新状态
	return s.db.DB.Model(&models.KnowledgeArticle{}).Where("id = ?", art.ID).
		Updates(map[string]interface{}{"status": "published", "updated_by": user}).Error
}

// List 获取知识库列表（分页）
func (s *KnowledgeService) List(page, pageSize int) ([]models.KnowledgeArticle, int64, error) {
	var items []models.KnowledgeArticle
	var total int64

	offset := (page - 1) * pageSize

	// 获取总数
	if err := s.db.DB.Model(&models.KnowledgeArticle{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	err := s.db.DB.Order("updated_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&items).Error

	return items, total, err
}

// GetByRule 获取规则名匹配的条目（支持模糊搜索，默认按更新时间倒序）
func (s *KnowledgeService) GetByRule(rule string, limit int) ([]models.KnowledgeArticle, error) {
	norm := s.Normalize(rule)
	if limit <= 0 {
		limit = 100 // 增加默认限制
	}
	var items []models.KnowledgeArticle
	// 使用 LIKE 进行模糊搜索
	err := s.db.DB.Where("alert_rule_name_normalized LIKE ?", "%"+norm+"%").
		Order("updated_at DESC").Limit(limit).Find(&items).Error
	return items, err
}

// GetManifest 读取并返回manifest原文
func (s *KnowledgeService) GetManifest(ctx context.Context, objectKey string) ([]byte, error) {
	return s.storage.Get(ctx, objectKey)
}

// PresignAssetPut 生成附件上传预签名URL
func (s *KnowledgeService) PresignAssetPut(ctx context.Context, articleID, filename, contentType string) (key string, url string, err error) {
	base := path.Join("knowledge", articleID, "assets")
	// 使用 uuid 生成唯一文件名，保留原后缀
	ext := ""
	if i := strings.LastIndex(filename, "."); i >= 0 {
		ext = filename[i:]
	}
	key = path.Join(base, uuid.NewString()+ext)
	u, e := s.storage.PresignPut(ctx, key, 10*time.Minute, contentType)
	if e != nil {
		return "", "", e
	}
	return key, u, nil
}

// GetByID 查询单条
func (s *KnowledgeService) GetByID(id uint64) (*models.KnowledgeArticle, error) {
	var art models.KnowledgeArticle
	if err := s.db.DB.First(&art, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &art, nil
}

// GetManifestByArticleID 读取 manifest（通过文章ID）
func (s *KnowledgeService) GetManifestByArticleID(ctx context.Context, id uint64) ([]byte, error) {
	art, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	return s.GetManifest(ctx, art.ObjectKey)
}

// Delete 删除知识条目（软删除）
func (s *KnowledgeService) Delete(ctx context.Context, id uint64) error {
	var art models.KnowledgeArticle
	if err := s.db.DB.First(&art, "id = ?", id).Error; err != nil {
		return err
	}

	// 软删除
	return s.db.DB.Delete(&art).Error
}

// BuildKnowledgeBaseForRule 获取指定规则名的知识库内容（优先markdown，其次转换tiptap JSON为markdown）
func (s *KnowledgeService) BuildKnowledgeBaseForRule(ctx context.Context, rule string) (string, error) {
	if s == nil {
		return "", fmt.Errorf("knowledge service is nil")
	}

	articles, err := s.GetByRule(rule, 1)
	if err != nil {
		return "", err
	}
	if len(articles) == 0 {
		return "", nil
	}

	manifestBytes, err := s.GetManifest(ctx, articles[0].ObjectKey)
	if err != nil {
		return "", err
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return "", err
	}

	// 优先使用已存储的 markdown
	if md, ok := manifest["markdown"].(string); ok && strings.TrimSpace(md) != "" {
		return md, nil
	}

	// 如果没有 markdown，尝试将 tiptap JSON 转换为 markdown
	if content, ok := manifest["content"].(map[string]interface{}); ok {
		if tiptap, ok := content["tiptap"]; ok {
			if tiptapMap, ok := tiptap.(map[string]interface{}); ok {
				md := tiptapToMarkdown(tiptapMap)
				if strings.TrimSpace(md) != "" {
					return md, nil
				}
			}
		}
	}

	return "", nil
}

// tiptapToMarkdown 将 Tiptap JSON 转换为 Markdown 格式
func tiptapToMarkdown(node map[string]interface{}) string {
	nodeType, _ := node["type"].(string)

	switch nodeType {
	case "doc":
		return processContent(node)

	case "paragraph":
		content := processContent(node)
		if content == "" {
			return "\n"
		}
		return content + "\n\n"

	case "heading":
		level := 1
		if attrs, ok := node["attrs"].(map[string]interface{}); ok {
			if l, ok := attrs["level"].(float64); ok {
				level = int(l)
			}
		}
		prefix := strings.Repeat("#", level)
		return prefix + " " + processContent(node) + "\n\n"

	case "bulletList":
		return processListItems(node, "- ")

	case "orderedList":
		return processListItems(node, "1. ")

	case "listItem":
		return processContent(node)

	case "codeBlock":
		language := ""
		if attrs, ok := node["attrs"].(map[string]interface{}); ok {
			if lang, ok := attrs["language"].(string); ok {
				language = lang
			}
		}
		code := processContent(node)
		return "```" + language + "\n" + code + "\n```\n\n"

	case "blockquote":
		lines := strings.Split(strings.TrimSpace(processContent(node)), "\n")
		for i, line := range lines {
			lines[i] = "> " + line
		}
		return strings.Join(lines, "\n") + "\n\n"

	case "horizontalRule":
		return "---\n\n"

	case "hardBreak":
		return "\n"

	case "text":
		text, _ := node["text"].(string)

		// 处理文本标记（加粗、斜体等）
		if marks, ok := node["marks"].([]interface{}); ok {
			for _, mark := range marks {
				if markMap, ok := mark.(map[string]interface{}); ok {
					markType, _ := markMap["type"].(string)
					switch markType {
					case "bold":
						text = "**" + text + "**"
					case "italic":
						text = "*" + text + "*"
					case "code":
						text = "`" + text + "`"
					case "strike":
						text = "~~" + text + "~~"
					case "link":
						if attrs, ok := markMap["attrs"].(map[string]interface{}); ok {
							if href, ok := attrs["href"].(string); ok {
								text = "[" + text + "](" + href + ")"
							}
						}
					}
				}
			}
		}

		return text

	default:
		// 对于未知类型，尝试处理其内容
		return processContent(node)
	}
}

// processContent 处理节点的 content 数组
func processContent(node map[string]interface{}) string {
	content, ok := node["content"].([]interface{})
	if !ok {
		return ""
	}

	var result strings.Builder
	for _, item := range content {
		if itemMap, ok := item.(map[string]interface{}); ok {
			result.WriteString(tiptapToMarkdown(itemMap))
		}
	}

	return result.String()
}

// processListItems 处理列表项
func processListItems(node map[string]interface{}, prefix string) string {
	content, ok := node["content"].([]interface{})
	if !ok {
		return ""
	}

	var result strings.Builder
	for _, item := range content {
		if itemMap, ok := item.(map[string]interface{}); ok {
			itemContent := processContent(itemMap)
			// 为每一行添加列表前缀
			lines := strings.Split(strings.TrimSpace(itemContent), "\n")
			for i, line := range lines {
				if i == 0 {
					result.WriteString(prefix + line + "\n")
				} else {
					result.WriteString("  " + line + "\n")
				}
			}
		}
	}
	result.WriteString("\n")

	return result.String()
}
