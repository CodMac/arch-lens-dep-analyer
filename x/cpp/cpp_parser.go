package cpp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	cppast "github.com/CodMac/arch-lens-dep-analyer/x/cpp/ast"
	sitter "github.com/tree-sitter/go-tree-sitter"
	treeSitterC "github.com/tree-sitter/tree-sitter-c/bindings/go"
	treeSitterCpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
)

// CppParserFactory 实现 core.ParserFactory，创建 C/C++ 语言解析器。
type CppParserFactory struct{}

// NewParser 创建一个 C/C++ parser 实例。
func (f *CppParserFactory) NewParser() core.Parser {
	return &cppParser{parser: sitter.NewParser()}
}

type cppParser struct {
	parser *sitter.Parser
}

// ParseFile 解析指定路径的 C/C++ 源文件。
func (p *cppParser) ParseFile(path string) (coreast.SourceFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return p.parseBytes(path, content)
}

// parseBytes 解析源码字节内容。
// 按扩展名选择 grammar：
//   - .c / .h 纯 C 优先用 c grammar（cpp 是 C 超集，.h 默认 cpp 以兼容 C++ 头文件，
//     但纯 C 项目 .c 用 c grammar 可获得与源语言一致的 AST）
//   - .cpp/.hpp/.cc/.cxx 用 cpp grammar
func (p *cppParser) parseBytes(path string, content []byte) (coreast.SourceFile, error) {
	lang := sitter.NewLanguage(languageForPath(path))
	if err := p.parser.SetLanguage(lang); err != nil {
		return nil, err
	}
	tree := p.parser.Parse(content, nil)
	if tree == nil {
		return nil, fmt.Errorf("tree-sitter failed to parse %s", path)
	}
	return cppast.NewSourceFile(path, content, tree), nil
}

// Close 释放 parser 底层资源。
func (p *cppParser) Close() {
	p.parser.Close()
}

// languageForPath 根据文件扩展名选择 tree-sitter grammar 指针。
func languageForPath(path string) unsafe.Pointer {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".c":
		return treeSitterC.Language()
	case ".cpp", ".hpp", ".cc", ".cxx", ".h":
		// .h 默认 cpp grammar（超集，兼容纯 C 与 C++ 头文件）；
		// 其余 C++ 扩展名用 cpp grammar。
		return treeSitterCpp.Language()
	default:
		// 未知扩展名默认 cpp（超集）。
		return treeSitterCpp.Language()
	}
}
