package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	sitter "github.com/tree-sitter/go-tree-sitter"
	// 导入所有语言绑定，确保 GetLanguage 可以找到
	_ "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

// Parser 定义了所有语言解析器的通用能力
type Parser interface {
	// ParseFile 的签名需要更新以适应配置参数
	ParseFile(filePath string, enableASTOutput bool, formatAST bool) (*sitter.Node, *[]byte, error)
	Close()
}

// TreeSitterParser Parser的具体实现
type TreeSitterParser struct {
	Language core.Language // 当前解析器针对的语言
	tsParser *sitter.Parser
}

// NewParser 创建一个新的 TreeSitterParser 实例
func NewParser(lang core.Language) (Parser, error) {
	tsLang, err := core.GetLanguage(lang)
	if err != nil {
		return nil, err
	}

	tsParser := sitter.NewParser()
	tsParser.SetLanguage(tsLang)

	return &TreeSitterParser{
		Language: lang,
		tsParser: tsParser,
	}, nil
}

// ParseFile 实现了 ParserInterface 接口
// 接受 enableASTOutput 和 formatAST 两个布尔参数
func (p *TreeSitterParser) ParseFile(filePath string, enableASTOutput bool, formatAST bool) (*sitter.Node, *[]byte, error) {
	// 1. 读取文件内容
	sourceBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// 2. 解析文件内容
	// 假设使用 UTF8 编码
	tree := p.tsParser.Parse(sourceBytes, nil)
	if tree == nil {
		return nil, nil, fmt.Errorf("tree-sitter failed to parse file %s", filePath)
	}

	// 3. 输出 AST 文件 (根据 enableASTOutput 和 formatAST 决定行为)
	if enableASTOutput {
		if err := writeASTToFile(tree.RootNode(), filePath, sourceBytes, formatAST); err != nil {
			// 警告而不是失败，不影响主流程
			fmt.Printf("[Warning] Failed to write AST file for %s: %v\n", filePath, err)
		}
	}

	// 4. 返回 AST 根节点
	return tree.RootNode(), &sourceBytes, nil
}

// Close 释放 Tree-sitter 内部资源 (可选)
func (p *TreeSitterParser) Close() {
	if p.tsParser != nil {
		p.tsParser.Close()
	}
}

// writeASTToFile 根据 formatFlag 决定使用紧凑或格式化的 S-expression
func writeASTToFile(rootNode *sitter.Node, filePath string, sourceBytes []byte, formatAST bool) error {
	var astString string

	// 是否格式化
	if formatAST {
		astString = formatSExpression(rootNode, sourceBytes, 0) // 使用格式化函数（缩进和换行）
	} else {
		astString = rootNode.ToSexp()
	}

	// 构建 AST 文件的输出路径
	dir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	astFileName := fileName + ".ast"
	if formatAST {
		astFileName += ".format"
	}
	astFilePath := filepath.Join(dir, astFileName)

	// 写入文件
	return os.WriteFile(astFilePath, []byte(astString), 0644)
}

// formatSExpression 递归遍历抽象语法树（AST）, 并生成包含节点字段、行列坐标、字节范围及文本特征的格式化S-expression字符串。
func formatSExpression(node *sitter.Node, sourceCode []byte, indentLevel int) string {
	indent := strings.Repeat("  ", indentLevel)
	var builder strings.Builder

	// 1. 组装节点基础头信息: 缩进 + (Kind
	builder.WriteString(fmt.Sprintf("%s(%s", indent, node.Kind()))

	// 2. 增强特征 A: 如果该节点在其父节点中拥有特定的 FieldName，将其标记暴露出来 (极度有利于调试 Extractor)
	if parent := node.Parent(); parent != nil {
		// 遍历父节点的所有子节点，匹配当前节点以找出其 field_name
		for i := 0; i < int(parent.ChildCount()); i++ {
			if node == parent.Child(uint(i)) {
				if fieldName := parent.FieldNameForChild(uint32(i)); fieldName != "" {
					builder.WriteString(fmt.Sprintf(" field=%s", fieldName))
				}
				break
			}
		}
	}

	// 3. 增强特征 B: 打印详尽的行列坐标与字节区间 [StartByte-EndByte, Line:Col-Line:Col]
	startPos := node.StartPosition()
	endPos := node.EndPosition()
	// tree-sitter 坐标从 0 开始，转换为人类及 IDE 常用的从 1 开始的坐标
	builder.WriteString(fmt.Sprintf(" loc=[%d-%d, %d:%d-%d:%d]",
		node.StartByte(), node.EndByte(),
		startPos.Row+1, startPos.Column+1,
		endPos.Row+1, endPos.Column+1,
	))

	// 4. 增强特征 C: 显式标记匿名节点(标点、关键字等)，方便一眼识破非命名节点
	if !node.IsNamed() {
		builder.WriteString(" anonymous")
	}

	// 5. 处理叶子节点 (没有子命名节点的节点)
	if node.NamedChildCount() == 0 {
		start := node.StartByte()
		end := node.EndByte()

		var content string
		if start < end && int(end) <= len(sourceCode) {
			content = string(sourceCode[start:end])
		}

		trimmedContent := strings.TrimSpace(content)
		if trimmedContent != "" {
			// 智能清洗叶子节点内容: 压缩换行符防止破坏 S-Expression 缩进结构
			cleanContent := strings.ReplaceAll(trimmedContent, "\n", "\\n")
			cleanContent = strings.ReplaceAll(cleanContent, "\r", "")
			if len(cleanContent) > 40 { // 对超长内容(如超长字符串、长注释)进行截断保护
				cleanContent = cleanContent[:37] + "..."
			}
			builder.WriteString(fmt.Sprintf(" text=%q)", cleanContent))
		} else {
			builder.WriteString(")")
		}
		return builder.String()
	}

	// 6. 递归处理包含子节点的复合节点 (深度优先遍历)
	builder.WriteString("\n")

	// 为了打印更完整的语法细节，建议改用 Child 遍历(包含非命名节点)而不仅是 NamedChild，
	// 这样可以看清所有逗号、括号和关键字所在的精准坐标位置。
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(uint(i))
		if child == nil {
			continue
		}

		// 过滤掉纯空白噪音的叶子标点，避免输出文件无意义暴涨，但保留关键的有位置的节点
		if !child.IsNamed() {
			start := child.StartByte()
			end := child.EndByte()
			if start < end && int(end) <= len(sourceCode) {
				txt := strings.TrimSpace(string(sourceCode[start:end]))
				if txt == "" || isPunctuation(txt) {
					// 如果是无业务意义的纯标点符号(且非关键字)，可以在 AST 输出中跳过
					continue
				}
			}
		}

		builder.WriteString(formatSExpression(child, sourceCode, indentLevel+1))
		builder.WriteString("\n")
	}

	// 7. 闭合当前节点
	result := builder.String()
	result = strings.TrimSuffix(result, "\n")

	builder.Reset()
	builder.WriteString(result)
	builder.WriteString(fmt.Sprintf("\n%s)", indent))

	return builder.String()
}

// isPunctuation 检查内容是否可能只是标点符号
func isPunctuation(s string) bool {
	for _, char := range s {
		if !strings.ContainsRune("(){}[];,\"'`", char) {
			return false
		}
	}
	return true
}
