package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// kiteEncryptKey 用于加密的密钥，与 Kite 的 KITE_ENCRYPT_KEY 保持一致
var kiteEncryptKey = "kite-default-encryption-key-change-in-production"

func init() {
	// 从环境变量加载加密密钥
	if key := os.Getenv("KITE_ENCRYPT_KEY"); key != "" {
		kiteEncryptKey = key
	}
}

// SetKiteEncryptKey 设置加密密钥
func SetKiteEncryptKey(key string) {
	if key != "" {
		kiteEncryptKey = key
	}
}

// KiteSecretString 与 Kite 兼容的加密字符串类型
// 使用 AES-256-GCM 加密，与 Kite 的 SecretString 完全兼容
type KiteSecretString string

// String 返回明文字符串
func (s KiteSecretString) String() string {
	return string(s)
}

// MarshalJSON 序列化为 JSON 时返回明文字符串
func (s KiteSecretString) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(s))
}

// EncryptForKite 使用 Kite 兼容的方式加密字符串
func EncryptForKite(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	keyHash := sha256.Sum256([]byte(kiteEncryptKey))
	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(input), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Scan implements the sql.Scanner interface (用于从数据库读取并解密)
func (s *KiteSecretString) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	var encryptedStr string
	switch v := value.(type) {
	case string:
		encryptedStr = v
	case []byte:
		encryptedStr = string(v)
	default:
		return fmt.Errorf("cannot scan %T into KiteSecretString", value)
	}
	if encryptedStr == "" {
		*s = ""
		return nil
	}
	// 解密
	decrypted, err := DecryptFromKite(encryptedStr)
	if err != nil {
		// 如果解密失败，可能是明文数据，直接使用
		*s = KiteSecretString(encryptedStr)
		return nil //nolint:nilerr // Intentional fallback to plaintext
	}
	*s = KiteSecretString(decrypted)
	return nil
}

// Value implements the driver.Valuer interface (用于写入数据库时自动加密)
func (s KiteSecretString) Value() (driver.Value, error) {
	if s == "" {
		return "", nil
	}
	encrypted, err := EncryptForKite(string(s))
	if err != nil {
		return nil, err
	}
	return encrypted, nil
}

// DecryptFromKite 解密 Kite 格式的加密字符串
func DecryptFromKite(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}

	keyHash := sha256.Sum256([]byte(kiteEncryptKey))
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}
