package helper

import (
	"regexp"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// =============================================================================
// 一些通用的方法
// =============================================================================

func IsScopeContainer(k model.ElementKind) bool {
	switch k {
	case model.Class, model.Interface, model.Enum, model.KAnnotation,
		model.Method, model.Lambda, model.ScopeBlock, model.AnonymousClass:
		return true
	}
	return false
}

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

// =============================================================================
// tree-sitter 节点相关
// =============================================================================

func GetNodeContent(n *sitter.Node, src []byte) string {
	if n == nil {
		return ""
	}
	return n.Utf8Text(src)
}

func FindNamedChildOfType(n *sitter.Node, nodeType string) *sitter.Node {
	for i := 0; i < int(n.NamedChildCount()); i++ {
		child := n.NamedChild(uint(i))
		if child.Kind() == nodeType {
			return child
		}
	}
	return nil
}

func FindNearestKind(n *sitter.Node, kinds ...string) *sitter.Node {
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

func ExtractLocation(n *sitter.Node, path string) *model.Location {
	return &model.Location{
		FilePath:    path,
		StartLine:   int(n.StartPosition().Row) + 1,
		EndLine:     int(n.EndPosition().Row) + 1,
		StartColumn: int(n.StartPosition().Column),
		EndColumn:   int(n.EndPosition().Column),
	}
}

func MatchLocation(n *sitter.Node, ele *model.CodeElement) bool {
	return (int(n.StartPosition().Row)+1 == ele.Location.StartLine) &&
		(int(n.EndPosition().Row)+1 == ele.Location.EndLine) &&
		(int(n.StartPosition().Column) == ele.Location.StartColumn) &&
		(int(n.EndPosition().Column) == ele.Location.EndColumn)
}

// =============================================================================
// metrics指标计算
// =============================================================================

func CalculateLOC(source []byte) int {
	// 1. 移除块注释
	reBlock := regexp.MustCompile(`(?s)/\*.*?\*/`)
	content := reBlock.ReplaceAll(source, []byte(""))

	// 2. 按行切分并统计
	lines := strings.Split(string(content), "\n")
	loc := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 排除空行和单行注释
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			loc++
		}
	}
	return loc
}

func CalculateRawLOC(source []byte) int {
	// 按行切分并统计
	lines := strings.Split(string(source), "\n")

	return len(lines)
}

func CalculateComplexity(node *sitter.Node) int {
	complexity := 1
	var traverse func(*sitter.Node)
	traverse = func(n *sitter.Node) {
		if n == nil {
			return
		}
		kind := n.Kind()
		switch kind {
		case "if_statement", "for_statement", "while_statement", "do_statement",
			"catch_clause", "conditional_expression", "ternary_expression", "switch_label":
			complexity++
		case "binary_expression":
			// 对于 binary_expression，我们需要检查运算符是否为 && 或 ||
			// 注意：tree-sitter-java 可能将 && 和 || 视为不同的子节点
			for i := 0; i < int(n.ChildCount()); i++ {
				child := n.Child(uint(i))
				if child.Kind() == "&&" || child.Kind() == "||" {
					complexity++
				}
			}
		}
		for i := 0; i < int(n.ChildCount()); i++ {
			traverse(n.Child(uint(i)))
		}
	}
	traverse(node)
	return complexity
}
