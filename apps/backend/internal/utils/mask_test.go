package utils

import (
	"testing"
)

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "普通邮箱",
			email:    "user@example.com",
			expected: "use***@example.com",
		},
		{
			name:     "短邮箱",
			email:    "ab@example.com",
			expected: "a***@example.com",
		},
		{
			name:     "长邮箱",
			email:    "verylongusername@example.com",
			expected: "ver***@example.com",
		},
		{
			name:     "CAS邮箱",
			email:    "casuser@cas.local",
			expected: "cas***@cas.local",
		},
		{
			name:     "空邮箱",
			email:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskEmail(tt.email)
			if result != tt.expected {
				t.Errorf("MaskEmail(%q) = %q, want %q", tt.email, result, tt.expected)
			}
		})
	}
}

func TestMaskUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		expected string
	}{
		{
			name:     "普通用户名",
			username: "johndoe",
			expected: "joh***",
		},
		{
			name:     "短用户名",
			username: "ab",
			expected: "a***",
		},
		{
			name:     "中等用户名",
			username: "user",
			expected: "us***",
		},
		{
			name:     "CAS用户",
			username: "casuser",
			expected: "cas***",
		},
		{
			name:     "单字符",
			username: "a",
			expected: "a***",
		},
		{
			name:     "空用户名",
			username: "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskUsername(tt.username)
			if result != tt.expected {
				t.Errorf("MaskUsername(%q) = %q, want %q", tt.username, result, tt.expected)
			}
		})
	}
}

func TestMaskName(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		expected string
	}{
		{
			name:     "中文两字姓名",
			fullName: "张三",
			expected: "张*",
		},
		{
			name:     "中文三字姓名",
			fullName: "李四五",
			expected: "李**",
		},
		{
			name:     "英文姓名",
			fullName: "John Doe",
			expected: "J*** D***",
		},
		{
			name:     "单字",
			fullName: "王",
			expected: "王",
		},
		{
			name:     "空姓名",
			fullName: "",
			expected: "",
		},
		{
			name:     "英文单名",
			fullName: "Alice",
			expected: "A**",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskName(tt.fullName)
			if result != tt.expected {
				t.Errorf("MaskName(%q) = %q, want %q", tt.fullName, result, tt.expected)
			}
		})
	}
}

func TestMaskUserInfo(t *testing.T) {
	username := "casuser"
	email := "casuser@cas.local"
	name := "张三"

	maskedUsername, maskedEmail, maskedName := MaskUserInfo(username, email, name)

	expectedUsername := "cas***"
	expectedEmail := "cas***@cas.local"
	expectedName := "张*"

	if maskedUsername != expectedUsername {
		t.Errorf("MaskUserInfo username = %q, want %q", maskedUsername, expectedUsername)
	}
	if maskedEmail != expectedEmail {
		t.Errorf("MaskUserInfo email = %q, want %q", maskedEmail, expectedEmail)
	}
	if maskedName != expectedName {
		t.Errorf("MaskUserInfo name = %q, want %q", maskedName, expectedName)
	}
}

func TestShouldMaskForUser(t *testing.T) {
	tests := []struct {
		name          string
		currentUserID string
		targetUserID  string
		isAdmin       bool
		expected      bool
	}{
		{
			name:          "查看自己的信息",
			currentUserID: "user-123",
			targetUserID:  "user-123",
			isAdmin:       false,
			expected:      false,
		},
		{
			name:          "管理员查看其他用户",
			currentUserID: "admin-123",
			targetUserID:  "user-456",
			isAdmin:       true,
			expected:      true,
		},
		{
			name:          "普通用户查看其他用户",
			currentUserID: "user-123",
			targetUserID:  "user-456",
			isAdmin:       false,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldMaskForUser(tt.currentUserID, tt.targetUserID, tt.isAdmin)
			if result != tt.expected {
				t.Errorf("ShouldMaskForUser() = %v, want %v", result, tt.expected)
			}
		})
	}
}
