package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// KnowledgeArticle 知识库文章（数据库仅保存 MinIO 主对象 key）
type KnowledgeArticle struct {
	BaseModel
	AlertRuleName           string         `json:"alert_rule_name" gorm:"type:varchar(255);not null"`
	AlertRuleNameNormalized string         `json:"alert_rule_name_normalized" gorm:"type:varchar(255);index;not null"`
	Tags                    datatypes.JSON `json:"tags" gorm:"type:jsonb"`
	Status                  string         `json:"status" gorm:"type:varchar(16);default:'draft'"`
	ObjectKey               string         `json:"object_key" gorm:"type:varchar(512);not null"`
	Version                 int            `json:"version" gorm:"not null;default:1"`
	CreatedBy               string         `json:"created_by" gorm:"type:varchar(128)"`
	UpdatedBy               string         `json:"updated_by" gorm:"type:varchar(128)"`
}

// KnowledgeArticleVersion 历史版本（可选）
type KnowledgeArticleVersion struct {
	BaseModel
	ArticleID     uuid.UUID `json:"article_id" gorm:"not null"`
	Version       int       `json:"version" gorm:"not null"`
	ObjectKey     string    `json:"object_key" gorm:"type:varchar(512);not null"`
	ChangeSummary string    `json:"change_summary" gorm:"type:text"`
	CreatedBy     string    `json:"created_by" gorm:"type:varchar(128)"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func (KnowledgeArticle) TableName() string        { return "knowledge_articles" }
func (KnowledgeArticleVersion) TableName() string { return "knowledge_article_versions" }
