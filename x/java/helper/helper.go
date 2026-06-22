package helper

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// =============================================================================
// 一些通用的方法
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
