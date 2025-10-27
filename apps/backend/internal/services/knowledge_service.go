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
	"robusta-web/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// KnowledgeService 知识库服务
type KnowledgeService struct {
	db      *db.Database
	storage PayloadStorage
}

func NewKnowledgeService(database *db.Database, storage PayloadStorage) *KnowledgeService {
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
	Title     string                   `json:"title"`
	Tags      []string                 `json:"tags,omitempty"`
	Status    string                   `json:"status"`
	Version   int                      `json:"version"`
	Excerpt   string                   `json:"excerpt,omitempty"`
	Content   map[string]interface{}   `json:"content"`
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
func (s *KnowledgeService) Create(ctx context.Context, alertRuleName, title string, tags []string, user string) (*models.KnowledgeArticle, error) {
	articleID := uuid.NewString()
	objectKey := s.manifestKey(articleID)

	norm := s.Normalize(alertRuleName)
	tagsJSON, _ := json.Marshal(tags)

	art := &models.KnowledgeArticle{
		BaseModel:               models.BaseModel{ID: uuid.New()},
		AlertRuleName:           alertRuleName,
		AlertRuleNameNormalized: norm,
		Title:                   title,
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
		Title:     title,
		Tags:      tags,
		Status:    art.Status,
		Version:   art.Version,
		Content:   map[string]interface{}{"tiptap": map[string]interface{}{"type": "doc", "content": []interface{}{}}, "markdown": ""},
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
func (s *KnowledgeService) Update(ctx context.Context, id string, payload Manifest, user string) (*models.KnowledgeArticle, error) {
	var art models.KnowledgeArticle
	if err := s.db.DB.First(&art, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// 合并关键信息
	art.Title = payload.Title
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
func (s *KnowledgeService) Publish(ctx context.Context, id string, changeSummary, user string) error {
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
		BaseModel:     models.BaseModel{ID: uuid.New()},
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

// GetByRule 获取规则名匹配的条目（默认按更新时间倒序）
func (s *KnowledgeService) GetByRule(rule string, limit int) ([]models.KnowledgeArticle, error) {
	norm := s.Normalize(rule)
	if limit <= 0 {
		limit = 100 // 增加默认限制
	}
	var items []models.KnowledgeArticle
	err := s.db.DB.Where("alert_rule_name_normalized = ?", norm).
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
func (s *KnowledgeService) GetByID(id string) (*models.KnowledgeArticle, error) {
	var art models.KnowledgeArticle
	if err := s.db.DB.First(&art, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &art, nil
}

// GetManifestByArticleID 读取 manifest（通过文章ID）
func (s *KnowledgeService) GetManifestByArticleID(ctx context.Context, id string) ([]byte, error) {
	art, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	return s.GetManifest(ctx, art.ObjectKey)
}

// Delete 删除知识条目（软删除）
func (s *KnowledgeService) Delete(ctx context.Context, id string) error {
	var art models.KnowledgeArticle
	if err := s.db.DB.First(&art, "id = ?", id).Error; err != nil {
		return err
	}
	
	// 软删除
	return s.db.DB.Delete(&art).Error
}
