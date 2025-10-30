package utils

import (
	"strings"
)

// MaskEmail 对邮箱地址进行掩码处理
// 示例: user@example.com -> u***@example.com
//
//	longusername@example.com -> lon***@example.com
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email // 无效的邮箱格式，返回原值
	}

	localPart := parts[0]
	domain := parts[1]

	// 如果本地部分长度小于等于3，只显示第一个字符
	if len(localPart) <= 3 {
		return string(localPart[0]) + "***@" + domain
	}

	// 显示前3个字符，其余用***代替
	return localPart[:3] + "***@" + domain
}

// MaskUsername 对用户名进行掩码处理
// 示例: johndoe -> joh***
//
//	ab -> a***
func MaskUsername(username string) string {
	if username == "" {
		return ""
	}

	length := len(username)

	// 如果用户名长度小于等于2，只显示第一个字符
	if length <= 2 {
		return string(username[0]) + "***"
	}

	// 如果用户名长度小于等于4，显示前2个字符
	if length <= 4 {
		return username[:2] + "***"
	}

	// 显示前3个字符，其余用***代替
	return username[:3] + "***"
}

// MaskName 对姓名进行掩码处理
// 中文姓名：显示姓氏，名字用*代替（如：张三 -> 张*）
// 英文姓名：显示首字母和末字母（如：John Doe -> J*** D***）
func MaskName(name string) string {
	if name == "" {
		return ""
	}

	// 检查是否包含空格（可能是英文姓名）
	if strings.Contains(name, " ") {
		parts := strings.Fields(name)
		masked := make([]string, len(parts))
		for i, part := range parts {
			if len(part) <= 1 {
				masked[i] = part
			} else if len(part) <= 3 {
				masked[i] = string(part[0]) + "***"
			} else {
				masked[i] = string(part[0]) + "***"
			}
		}
		return strings.Join(masked, " ")
	}

	// 中文姓名或单个单词
	runes := []rune(name)
	length := len(runes)

	if length <= 1 {
		return name
	}

	if length == 2 {
		// 两个字的姓名，显示第一个字
		return string(runes[0]) + "*"
	}

	// 三个字及以上，显示第一个字
	return string(runes[0]) + "**"
}

// MaskUserInfo 对用户信息进行掩码处理
// 返回掩码后的用户名、邮箱和姓名
func MaskUserInfo(username, email, name string) (maskedUsername, maskedEmail, maskedName string) {
	return MaskUsername(username), MaskEmail(email), MaskName(name)
}

// ShouldMaskForUser 判断是否需要对用户信息进行掩码
// 如果是查看自己的信息，不需要掩码
// 如果是管理员查看其他用户，需要掩码
func ShouldMaskForUser(currentUserID, targetUserID string, isAdmin bool) bool {
	// 如果是查看自己的信息，不需要掩码
	if currentUserID == targetUserID {
		return false
	}

	// 管理员查看其他用户时，需要掩码
	return isAdmin
}
