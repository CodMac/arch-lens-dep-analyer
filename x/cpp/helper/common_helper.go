// Package helper 提供 C/C++ 语言的通用工具函数（QN 拼接/切分、文本清理等）。
// 注意：C++ 的 QN 分隔符是 ::，不能复用 Java helper 的点号硬编码逻辑，独立实现。
package helper

import (
	"strings"

	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// QNSep C++ 的 QN 分隔符。
const QNSep = "::"

// BuildQN 用 :: 拼接父 QN 与子符号名称为完整 QN。
func BuildQN(parentQN, name string) string {
	if parentQN == "" || parentQN == "::" {
		return name
	}
	return parentQN + QNSep + name
}

// SplitQN 将 QN 按 :: 切分为片段（保留原始顺序）。
func SplitQN(qn string) []string {
	return strings.Split(qn, QNSep)
}

// GetNodeContent 返回节点覆盖的源码文本。
func GetNodeContent(n coreast.Node, src []byte) string {
	if n == nil {
		return ""
	}
	return strings.TrimSpace(n.Utf8Text(src))
}

// ExtractLocation 从节点提取 Location。
func ExtractLocation(node coreast.Node, path string) *model.Location {
	if node == nil {
		return nil
	}
	sp, ep := node.StartPoint(), node.EndPoint()
	return &model.Location{
		FilePath:    path,
		StartLine:   int(sp.Row) + 1,
		EndLine:     int(ep.Row) + 1,
		StartColumn: int(sp.Column) + 1,
		EndColumn:   int(ep.Column) + 1,
	}
}

// IsPrimitive 判断类型名是否为 C/C++ 基本类型。
func IsPrimitive(t string) bool {
	switch t {
	case "void", "int", "long", "short", "char", "unsigned", "signed",
		"float", "double", "bool", "wchar_t", "size_t", "ssize_t", "int8_t",
		"int16_t", "int32_t", "int64_t", "uint8_t", "uint16_t", "uint32_t", "uint64_t",
		// auto 是类型占位符（推导类型），不是真实类型依赖。
		"auto":
		return true
	}
	return false
}

// StripArraySuffix 去掉类型文本的数组后缀：Foo[N] -> Foo。
func StripArraySuffix(s string) string {
	if i := strings.Index(s, "["); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// StripTemplateArgs 去掉类型文本的模板实参：Foo<int> -> Foo。
func StripTemplateArgs(s string) string {
	start := strings.Index(s, "<")
	if start < 0 {
		return s
	}
	// 从后往前找匹配的 >，避免嵌套模板误截断
	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '<':
			depth++
		case '>':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[:start])
			}
		}
	}
	return strings.TrimSpace(s[:start])
}

// PkgOf 返回 QN 的命名空间前缀（按 :: 切分，去掉最后一段）。
// 全局符号（无 ::）返回空。
func PkgOf(qn string) string {
	if i := strings.LastIndex(qn, QNSep); i > 0 {
		return qn[:i]
	}
	return ""
}

// IsTypeVariable 判断类型名是否为模板类型变量（单大写字母，如 T/E/S/K/V/U）。
// 这类符号不是真实类型依赖，解析时应跳过。
func IsTypeVariable(name string) bool {
	if len(name) != 1 {
		return false
	}
	c := name[0]
	return c >= 'A' && c <= 'Z'
}

// NewExternalElement 构造 C/C++ 外部符号（IsFormExternal=true）。
// PackageName 按 :: 拆分取前缀；无 :: 归 <external>。
// 不复用 model.NewExternalElement（其按点号拆分，C++ QN 会拆错）。
func NewExternalElement(kind model.ElementKind, name, qualifiedName string) *model.CodeElement {
	pkg := "<external>"
	if i := strings.LastIndex(qualifiedName, QNSep); i > 0 {
		pkg = qualifiedName[:i]
	}
	return &model.CodeElement{
		Kind:           kind,
		Name:           name,
		QualifiedName:  qualifiedName,
		PackageName:    pkg,
		IsFormExternal: true,
	}
}

// NormalizeTypeText 归一化类型文本（去多余空白、* & 前空格）。
func NormalizeTypeText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " *", "*")
	s = strings.ReplaceAll(s, " &", "&")
	return strings.Join(strings.Fields(s), " ")
}

// BaseTypeName 从类型文本提取可解析的类型名：
// 剥指针/引用/const/数组后缀/模板实参，保留 :: 限定。
// 例：const std::vector<int>& -> std::vector；Foo*[10] -> Foo
func BaseTypeName(s string) string {
	s = NormalizeTypeText(s)
	// 去掉 C 风格的类型关键字前缀：struct Node / enum Color / union U / class C
	for _, kw := range []string{"struct ", "class ", "enum ", "union "} {
		s = strings.TrimPrefix(s, kw)
	}
	s = StripArraySuffix(s)
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "&", "")
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "const ")
	s = strings.TrimSuffix(s, " const")
	s = strings.TrimSpace(s)
	s = StripTemplateArgs(s)
	return strings.TrimSpace(s)
}
