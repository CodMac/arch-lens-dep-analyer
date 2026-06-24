package helper

import (
	"strings"
)

// =============================================================================
// 通用方法
// =============================================================================

func IsPotentialClassName(s string) bool {
	if s == "" || s == "this" || s == "super" {
		return false
	}
	if strings.Contains(s, "(") {
		return false
	}
	parts := strings.Split(s, ".")
	last := parts[len(parts)-1]
	if len(last) > 0 && last[0] >= 'A' && last[0] <= 'Z' {
		return true
	}
	return false
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
