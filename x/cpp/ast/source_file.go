package ast

import (
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// SourceFile 包装 tree-sitter 语法树，实现 core/ast.SourceFile 接口。
type SourceFile struct {
	path    string
	content []byte
	tree    *sitter.Tree
}

// NewSourceFile 创建 SourceFile 适配器。
func NewSourceFile(path string, content []byte, tree *sitter.Tree) *SourceFile {
	return &SourceFile{path: path, content: content, tree: tree}
}

// Root 返回语法树根节点。
func (sf *SourceFile) Root() coreast.Node {
	if sf.tree == nil {
		return nil
	}
	return Wrap(sf.tree.RootNode())
}

// Content 返回源码字节内容。
func (sf *SourceFile) Content() []byte { return sf.content }

// Path 返回文件路径。
func (sf *SourceFile) Path() string { return sf.path }

// Close 释放语法树资源。
func (sf *SourceFile) Close() {
	if sf.tree != nil {
		sf.tree.Close()
		sf.tree = nil
	}
}
