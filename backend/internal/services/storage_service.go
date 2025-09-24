package services

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"robusta-web/backend/internal/config"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// PayloadStorage 用于存储原始大文本或JSON数据的抽象接口
type PayloadStorage interface {
	Save(ctx context.Context, prefix string, data []byte, contentType string) (string, error)
}

// ObjectStorageService 基于MinIO的对象存储实现
type ObjectStorageService struct {
	client *minio.Client
	bucket string
}

// NewObjectStorageService 创建对象存储服务实例并确保目标Bucket存在
func NewObjectStorageService(cfg *config.Config) (*ObjectStorageService, error) {
	client, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化MinIO客户端失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := client.MakeBucket(ctx, cfg.MinIOBucketName, minio.MakeBucketOptions{}); err != nil {
		exists, errBucketExists := client.BucketExists(ctx, cfg.MinIOBucketName)
		if errBucketExists != nil {
			return nil, fmt.Errorf("检查MinIO存储桶失败: %w", errBucketExists)
		}
		if !exists {
			return nil, fmt.Errorf("创建MinIO存储桶失败: %w", err)
		}
	}

	return &ObjectStorageService{client: client, bucket: cfg.MinIOBucketName}, nil
}

// Save 保存对象并返回生成的对象键
func (s *ObjectStorageService) Save(ctx context.Context, prefix string, data []byte, contentType string) (string, error) {
	if len(data) == 0 {
		return "", nil
	}

	cleanedPrefix := strings.TrimSpace(prefix)
	cleanedPrefix = strings.Trim(cleanedPrefix, "/")
	if cleanedPrefix != "" {
		cleanedPrefix = path.Clean(cleanedPrefix)
		cleanedPrefix = strings.Trim(cleanedPrefix, "/")
	}

	objectID := uuid.NewString()
	objectKey := objectID
	if cleanedPrefix != "" {
		objectKey = fmt.Sprintf("%s/%s", cleanedPrefix, objectID)
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	reader := bytes.NewReader(data)
	uploadCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := s.client.PutObject(uploadCtx, s.bucket, objectKey, reader, int64(len(data)), minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("上传对象到MinIO失败: %w", err)
	}

	return objectKey, nil
}
