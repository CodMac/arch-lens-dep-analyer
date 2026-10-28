// Package helper 提供 Java 语言的通用解析辅助能力。
package helper

import (
	"strings"

	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// GetNodeContent 返回 AST 节点的源码文本（nil 安全）。
func GetNodeContent(n coreast.Node, src []byte) string {
	if n == nil {
		return ""
	}
	return n.Utf8Text(src)
}

// FindNearestKind 自 n 起向上查找第一个匹配 kind 的祖先节点（含自身），
// 遇到语句边界或 class_body 即停。
func FindNearestKind(n coreast.Node, kinds ...string) coreast.Node {
	for curr := n; curr != nil; curr = curr.Parent() {
		for _, k := range kinds {
			if curr.Kind() == k {
				return curr
			}
		}
		if strings.HasSuffix(curr.Kind(), "_statement") || curr.Kind() == "class_body" {
			break
		}
	}
	return nil
}

// MatchLocation 判断 AST 节点位置是否与元素位置一致。
func MatchLocation(n coreast.Node, ele *model.CodeElement) bool {
	if n == nil || ele == nil || ele.Location == nil {
		return false
	}
	sp := n.StartPoint()
	ep := n.EndPoint()
	return (sp.Row+1 == ele.Location.StartLine) &&
		(ep.Row+1 == ele.Location.EndLine) &&
		(sp.Column == ele.Location.StartColumn) &&
		(ep.Column == ele.Location.EndColumn)
}

// IsPotentialClassName 判断字符串是否可能是类名（首字母大写且非 this/super）。
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

// Contains 判断字符串切片是否包含目标。
func Contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// Clean 清洗类型/符号文本：去掉 new/extends/super 前缀、泛型、参数括号、数组尾缀等。
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

// IsPrimitiveType 判断是否为 Java 基本类型。
func IsPrimitiveType(t string) bool {
	pt := []string{"int", "long", "double", "float", "boolean", "char", "byte", "short", "void"}
	return Contains(pt, t)
}

// IsBoxedPrimitive 判断是否为基本类型的包装类短名。
func IsBoxedPrimitive(t string) bool {
	bt := []string{"Integer", "Long", "Double", "Float", "Boolean", "Character", "Byte", "Short", "Void"}
	return Contains(bt, t)
}

// HasMethodParameterInfo 从方法元素的 Mores 中读取参数类型 QN 列表。
func MethodParamsQN(elem interface{ GetMores() map[string]interface{} }) []string {
	return nil // 占位，实际使用处直接访问 Mores
}

// ExtractMoresString 从 Mores 字典安全读取字符串值。
func ExtractMoresString(mores map[string]interface{}, key string) string {
	if mores == nil {
		return ""
	}
	if v, ok := mores[key].(string); ok {
		return v
	}
	return ""
}

// ExtractMoresStrings 从 Mores 字典安全读取字符串切片值。
func ExtractMoresStrings(mores map[string]interface{}, key string) []string {
	if mores == nil {
		return nil
	}
	if v, ok := mores[key].([]string); ok {
		return v
	}
	return nil
}

// ExtractMoresBool 从 Mores 字典安全读取布尔值。
func ExtractMoresBool(mores map[string]interface{}, key string) bool {
	if mores == nil {
		return false
	}
	if v, ok := mores[key].(bool); ok {
		return v
	}
	return false
}
