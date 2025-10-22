package services

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
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
	Get(ctx context.Context, key string) ([]byte, error)
}

// ObjectStorageService 基于MinIO的对象存储实现
type ObjectStorageService struct {
	client *minio.Client
	bucket string
}

// NewObjectStorageService 创建对象存储服务实例并确保目标Bucket存在
func NewObjectStorageService(cfg *config.Config) (*ObjectStorageService, error) {
	endpoint := strings.TrimSpace(cfg.MinIO.Endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("MinIO服务未配置，请在配置文件或环境变量中设置 MINIO_ENDPOINT")
	}

	if strings.Contains(endpoint, "://") {
		parsed, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("解析MinIO地址失败: %w", err)
		}
		endpoint = parsed.Host
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, ""),
		Secure: cfg.MinIO.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化MinIO客户端失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := client.MakeBucket(ctx, cfg.MinIO.BucketName, minio.MakeBucketOptions{}); err != nil {
		exists, errBucketExists := client.BucketExists(ctx, cfg.MinIO.BucketName)
		if errBucketExists != nil {
			return nil, fmt.Errorf("检查MinIO存储桶失败: %w", errBucketExists)
		}
		if !exists {
			return nil, fmt.Errorf("创建MinIO存储桶失败: %w", err)
		}
	}

	return &ObjectStorageService{client: client, bucket: cfg.MinIO.BucketName}, nil
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

// Get 从MinIO获取对象数据
func (s *ObjectStorageService) Get(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, fmt.Errorf("对象键不能为空")
	}

	getCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	object, err := s.client.GetObject(getCtx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("从MinIO获取对象失败: %w", err)
	}
	defer object.Close()

	// 读取对象内容
	var buffer bytes.Buffer
	_, err = buffer.ReadFrom(object)
	if err != nil {
		return nil, fmt.Errorf("读取对象内容失败: %w", err)
	}

	return buffer.Bytes(), nil
}
