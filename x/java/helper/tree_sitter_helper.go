package helper

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// =============================================================================
// 【tree-sitter节点】 通用方法
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

// FindNamedChildren 查找指定类型的命名子节点
func FindNamedChildren(container *sitter.Node, nodeType string) []*sitter.Node {
	var result []*sitter.Node
	count := container.NamedChildCount()
	for i := uint(0); i < count; i++ {
		child := container.NamedChild(i)
		if child != nil && child.Kind() == nodeType {
			result = append(result, child)
		}
	}

	return result
}

func IsNodeContained(container, node *sitter.Node) bool {
	if container == nil || node == nil {
		return false
	}
	return container.StartByte() <= node.StartByte() && node.EndByte() <= container.EndByte()
}

// =============================================================================
// 【tree-sitter节点】 Location相关方法
// =============================================================================

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
