package helper

import (
	"regexp"
	"strings"

	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
)

// =============================================================================
// 【metrics指标计算】 通用方法
// =============================================================================

// CalculateLOC 计算有效代码行数（移除块注释、空行与单行注释）。
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

// CalculateRawLOC 计算原始行数。
func CalculateRawLOC(source []byte) int {
	lines := strings.Split(string(source), "\n")
	return len(lines)
}

// CalculateComplexity 计算圈复杂度（接受语言无关的 coreast.Node）。
func CalculateComplexity(node coreast.Node) int {
	if node == nil {
		return 0
	}
	complexity := 1
	var traverse func(coreast.Node)
	traverse = func(n coreast.Node) {
		if n == nil {
			return
		}
		kind := n.Kind()
		switch kind {
		case "if_statement", "for_statement", "while_statement", "do_statement", "enhanced_for_statement",
			"catch_clause", "conditional_expression", "ternary_expression", "switch_label":
			complexity++
		case "binary_expression":
			// 检查运算符是否为 && 或 ||（tree-sitter-java 以匿名子节点呈现运算符）。
			for i := 0; i < n.ChildCount(); i++ {
				child := n.Child(i)
				if child != nil && (child.Kind() == "&&" || child.Kind() == "||") {
					complexity++
				}
			}
		}
		for i := 0; i < n.ChildCount(); i++ {
			traverse(n.Child(i))
		}
	}
	traverse(node)
	return complexity
}
