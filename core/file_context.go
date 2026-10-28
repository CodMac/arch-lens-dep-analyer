package core

import (
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// ImportInfo 描述文件中的一个导入声明。
type ImportInfo struct {
	// RawImportPath 导入的原始路径（如 "java.util.List"）。
	RawImportPath string

	// Alias 导入的短名（如 "List"）。
	Alias string

	// Kind 导入声明的目标元素类型（Class/Constant/Package）。
	Kind model.ElementKind

	// IsStatic 是否为静态导入。
	IsStatic bool

	// IsWildcard 是否为通配符导入（如 java.util.*）。
	IsWildcard bool

	// Location 导入声明位置。
	Location *model.Location
}

// FileContext 描述单个源文件的符号定义集合与索引。
type FileContext struct {
	// FilePath 文件路径（相对项目根）。
	FilePath string

	// PackageName 包名。
	PackageName string

	// Root 语法树根节点（供 query 遍历/提取器使用）。
	Root coreast.Node

	// Definitions 本文件收集的定义（含包元素、文件元素、类/方法/字段/变量）。
	Definitions []*DefinitionEntry

	// Imports 导入表：短名 -> 导入信息列表（含通配符）。
	Imports map[string][]*ImportInfo

	// SourceBytes 源文件字节内容（供 AST 文本切片）。
	SourceBytes *[]byte

	// 索引：短名 -> 定义（文件内消歧）。
	shortNameMap map[string][]*DefinitionEntry

	// 索引：限定名 -> 定义。
	qualifiedNameMap map[string]*DefinitionEntry

	// 索引：元素 kind -> 定义。
	kindMap map[model.ElementKind][]*DefinitionEntry
}

// NewFileContext 创建一个空文件上下文。
func NewFileContext(filePath, packageName string, root coreast.Node, sourceBytes *[]byte) *FileContext {
	return &FileContext{
		FilePath:         filePath,
		PackageName:      packageName,
		Root:             root,
		Imports:          make(map[string][]*ImportInfo),
		SourceBytes:      sourceBytes,
		Definitions:      make([]*DefinitionEntry, 0),
		shortNameMap:     make(map[string][]*DefinitionEntry),
		qualifiedNameMap: make(map[string]*DefinitionEntry),
		kindMap:          make(map[model.ElementKind][]*DefinitionEntry),
	}
}

// AddDefinition 注册一个定义并更新索引。
func (fc *FileContext) AddDefinition(def *DefinitionEntry) {
	fc.Definitions = append(fc.Definitions, def)
	fc.shortNameMap[def.Element.Name] = append(fc.shortNameMap[def.Element.Name], def)
	fc.qualifiedNameMap[def.Element.QualifiedName] = def
	fc.kindMap[def.Element.Kind] = append(fc.kindMap[def.Element.Kind], def)
}

// AddImport 注册一个导入。
func (fc *FileContext) AddImport(shortName string, info *ImportInfo) {
	fc.Imports[shortName] = append(fc.Imports[shortName], info)
}

// FindByShortName 按短名返回定义（可能多个）。
func (fc *FileContext) FindByShortName(name string) ([]*DefinitionEntry, bool) {
	entries, ok := fc.shortNameMap[name]
	return entries, ok
}

// FindByQualifiedName 按限定名返回定义。
func (fc *FileContext) FindByQualifiedName(qn string) (*DefinitionEntry, bool) {
	entry, ok := fc.qualifiedNameMap[qn]
	return entry, ok
}

// FindByKind 按元素类型返回定义。
func (fc *FileContext) FindByKind(kind model.ElementKind) ([]*DefinitionEntry, bool) {
	entries, ok := fc.kindMap[kind]
	return entries, ok
}
