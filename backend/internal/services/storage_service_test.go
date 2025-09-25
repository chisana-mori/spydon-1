package services

import (
	"context"
	"io"
	"testing"
	"time"

	"robusta-web/backend/internal/config"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

// TestObjectStorageService_SaveAndRead 验证对象存储写入后可以正常读取
func TestObjectStorageService_SaveAndRead(t *testing.T) {
	cfg := &config.Config{
		MinIOEndpoint:   "localhost:9000",
		MinIOAccessKey:  "minioadmin",
		MinIOSecretKey:  "minioadmin",
		MinIOBucketName: "robusta-artifacts",
		MinIOUseSSL:     false,
	}

	service, err := NewObjectStorageService(cfg)
	if err != nil {
		t.Skipf("跳过测试，无法连接MinIO: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	payload := []byte(`{"message":"hello-minio"}`)
	key, err := service.Save(ctx, "unittest", payload, "application/json")
	require.NoError(t, err)
	require.NotEmpty(t, key)

	object, err := service.client.GetObject(ctx, service.bucket, key, minio.GetObjectOptions{})
	require.NoError(t, err)
	defer object.Close()

	data, err := io.ReadAll(object)
	require.NoError(t, err)
	require.Equal(t, payload, data)

	err = service.client.RemoveObject(ctx, service.bucket, key, minio.RemoveObjectOptions{})
	require.NoError(t, err)
}
