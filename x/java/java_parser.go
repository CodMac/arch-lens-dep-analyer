package java

import (
	"fmt"
	"os"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	javast "github.com/CodMac/arch-lens-dep-analyer/x/java/ast"
	sitter "github.com/tree-sitter/go-tree-sitter"
	treeSitterJava "github.com/tree-sitter/tree-sitter-java/bindings/go"
)

// JavaParserFactory 实现 core.ParserFactory，创建 Java 语言解析器。
type JavaParserFactory struct{}

// NewParser 创建一个 Java parser 实例。
func (f *JavaParserFactory) NewParser() core.Parser {
	return &javaParser{parser: sitter.NewParser()}
}

type javaParser struct {
	parser *sitter.Parser
}

// ParseFile 解析指定路径的 Java 源文件。
func (p *javaParser) ParseFile(path string) (coreast.SourceFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p.parseBytes(path, content)
}

// parseBytes 解析源码字节内容（供测试与内存解析复用）。
func (p *javaParser) parseBytes(path string, content []byte) (coreast.SourceFile, error) {
	lang := sitter.NewLanguage(treeSitterJava.Language())
	if err := p.parser.SetLanguage(lang); err != nil {
		return nil, err
	}
	tree := p.parser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("tree-sitter failed to parse %s", path)
	}
	return javast.NewSourceFile(path, content, tree), nil
}

// Close 释放 parser 底层资源。
func (p *javaParser) Close() {
	p.parser.Close()
}
