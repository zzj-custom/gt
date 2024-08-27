package template

import (
	"strings"
)

// ToLowerCamelCase 将字符串转换为小驼峰命名法
func ToLowerCamelCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	if len(parts) == 0 {
		return s
	}
	result := parts[0]
	for _, part := range parts[1:] {
		if part != "" {
			result += strings.Title(part)
		}
	}
	return result
}

// ToUpperCamelCase 将字符串转换为大驼峰命名法
func ToUpperCamelCase(s string) string {
	lowerCamel := ToLowerCamelCase(s)
	if len(lowerCamel) > 0 {
		return strings.Title(lowerCamel)
	}
	return lowerCamel
}

// GetInitials 获取大驼峰的首字母
func GetInitials(s string) string {
	var initials string
	words := strings.FieldsFunc(s, func(r rune) bool { return r == ' ' || r == '-' || r == '_' })
	for _, word := range words {
		if len(word) > 0 {
			initials += string(word[0])
		}
	}
	return initials
}
