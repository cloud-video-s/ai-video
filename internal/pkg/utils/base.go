package utils

import (
	"errors"
	"strings"
	"unicode/utf8"
)

func DesensitizeEmail(email string) (string, error) {
	// 按 "@" 分割，标准邮箱应只有一个 "@"
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", errors.New("invalid email format: missing '@' or empty username/domain")
	}

	username := parts[0]
	domain := parts[1]

	// 将用户名转为 rune 切片，以正确处理 Unicode 字符
	runes := []rune(username)
	n := utf8.RuneCountInString(username)

	var maskedUsername string
	if n <= 3 {
		maskedUsername = "***"
	} else {
		// 保留首尾字符，中间用 *** 替换
		maskedUsername = string(runes[0]) + "***" + string(runes[n-1])
	}

	return maskedUsername + "@" + domain, nil
}
