// Package ast 提供 Java 语言的 tree-sitter 节点到 core/ast 接口的适配。
package ast

import (
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Node 包装 *sitter.Node，实现 core/ast.Node 接口。
type Node struct {
	n *sitter.Node
}

// Wrap 将一个 tree-sitter 节点包装为 core/ast.Node。
func Wrap(n *sitter.Node) coreast.Node {
	if n == nil {
		return nil
	}
	return &Node{n: n}
}

// Unwrap 还原为底层 tree-sitter 节点。
func Unwrap(n coreast.Node) *sitter.Node {
	if n == nil {
		return nil
	}
	if an, ok := n.(*Node); ok {
		return an.n
	}
	return nil
}

// Kind 返回节点语法类型名。
func (n *Node) Kind() string { return n.n.Kind() }

// Utf8Text 返回节点源码文本。
func (n *Node) Utf8Text(src []byte) string { return n.n.Utf8Text(src) }

// ChildByFieldName 按字段名返回子节点。
func (n *Node) ChildByFieldName(name string) coreast.Node {
	return Wrap(n.n.ChildByFieldName(name))
}

// ChildByFieldNameRecursive 递归向下查找具名字段节点。
func (n *Node) ChildByFieldNameRecursive(name string) coreast.Node {
	for i := 0; i < int(n.n.NamedChildCount()); i++ {
		child := n.n.NamedChild(uint(i))
		if child.Kind() == name {
			return Wrap(child)
		}
		if rec := Wrap(child).ChildByFieldNameRecursive(name); rec != nil {
			return rec
		}
	}
	return nil
}

// NamedChild 返回第 i 个具名子节点。
func (n *Node) NamedChild(i int) coreast.Node {
	return Wrap(n.n.NamedChild(uint(i)))
}

// NamedChildCount 返回具名子节点数量。
func (n *Node) NamedChildCount() int { return int(n.n.NamedChildCount()) }

// Child 返回第 i 个子节点。
func (n *Node) Child(i int) coreast.Node {
	return Wrap(n.n.Child(uint(i)))
}

// ChildCount 返回子节点总数。
func (n *Node) ChildCount() int { return int(n.n.ChildCount()) }

// Parent 返回父节点。
func (n *Node) Parent() coreast.Node { return Wrap(n.n.Parent()) }

// PrevSibling 返回前一个具名兄弟节点（tree-sitter 的 PrevNamedSibling）。
func (n *Node) PrevSibling() coreast.Node { return Wrap(n.n.PrevNamedSibling()) }

// StartByte 返回起始字节偏移。
func (n *Node) StartByte() int { return int(n.n.StartByte()) }

// EndByte 返回结束字节偏移。
func (n *Node) EndByte() int { return int(n.n.EndByte()) }

// StartPoint 返回起始位置。
func (n *Node) StartPoint() coreast.Point {
	p := n.n.StartPosition()
	return coreast.Point{Row: int(p.Row), Column: int(p.Column)}
}

// EndPoint 返回结束位置。
func (n *Node) EndPoint() coreast.Point {
	p := n.n.EndPosition()
	return coreast.Point{Row: int(p.Row), Column: int(p.Column)}
}

// IsNamed 返回是否为具名节点。
func (n *Node) IsNamed() bool { return n.n.IsNamed() }

// IsError 返回是否为错误节点。
func (n *Node) IsError() bool { return n.n.IsError() }

// Equal 判断是否指向同一语法节点。
// 不能直接用结构体 == 比较（tree-sitter 的 TSNode 含 C 指针，两次取值会产生不同副本，== 恒为 false）。
// 改用节点在语法树内的唯一 Id 比较（同一棵树内 Id 唯一）。
func (n *Node) Equal(other coreast.Node) bool {
	o, ok := other.(*Node)
	if !ok {
		return false
	}
	return n.n.Id() == o.n.Id()
}

// String 返回节点描述。
func (n *Node) String() string {
	return "Node(" + n.n.Kind() + ")"
}
