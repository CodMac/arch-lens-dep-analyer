package helper

import (
	"strings"
	"unicode"
)

// =============================================================================
// 通用方法
// =============================================================================

// IsPotentialClassName 启发式判断一个标识符是否可能是类名（排除常量）
func IsPotentialClassName(name string) bool {
	if len(name) == 0 {
		return false
	}

	// 1. 类名首字母必须是大写字母或符合 Java 标识符起首
	runes := []rune(name)
	if !unicode.IsUpper(runes[0]) {
		return false
	}

	// 2. 区分全大写的常量 (如 MAP_SIZE, CONTEXT) 与 类名 (如 MapSize, Context)
	// 如果字符串中包含至少一个小写字母，说明它不是标准的纯大写常量，更偏向于类名
	hasLower := false
	for _, r := range runes {
		if unicode.IsLower(r) {
			hasLower = true
			break
		}
	}

	// 如果全是大写字母和数字/下划线，没有一个小写字母，则判定为常量而非类名
	return hasLower
}

func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func Clean(s string) string {
	s = strings.TrimPrefix(s, "@")
	s = strings.TrimPrefix(s, "new ")
	if strings.Contains(s, "extends ") {
		s = strings.Split(s, "extends ")[1]
	}
	if strings.Contains(s, "super ") {
		s = strings.Split(s, "super ")[1]
	}
	s = strings.Split(s, "<")[0]
	s = strings.Split(s, "(")[0]
	s = strings.TrimSuffix(s, "...")
	return strings.TrimSpace(strings.TrimRight(s, "> ,[]"))
}
