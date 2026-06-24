package helper

import (
	"regexp"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// =============================================================================
// 【metrics指标计算】 通用方法
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
