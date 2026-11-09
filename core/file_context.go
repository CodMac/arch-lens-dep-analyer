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

	// IsSystem 是否为系统/三方头文件（C/C++ #include <...> 而非 "..."）。
	IsSystem bool

	// Location 导入声明位置。
	Location *model.Location
}

// FileContext 描述单个源文件的符号定义集合与索引。
//
// 接口化设计（2026-11-06）：core 只保留契约，通用索引/去重机制由 BaseFileContext 提供，
// 语言实现内嵌 Base 复用并在需要处覆盖/扩展（如 C++ 的 namespace 列表）。
type FileContext interface {
	// FilePath 文件路径（相对项目根）。
	FilePath() string

	// SetFilePath 设置文件路径（引擎路径相对化时调用）。
	SetFilePath(path string)

	// PackageName 包名（Java/Go 单包语义；C++ 置空改用 Namespaces）。
	PackageName() string

	// SetPackageName 设置包名。
	SetPackageName(name string)

	// Namespaces 命名空间列表（C++ 一文件可含多个 namespace；其余语言为空）。
	Namespaces() []string

	// SetNamespaces 设置命名空间列表。
	SetNamespaces(ns []string)

	// Root 语法树根节点（供 query 遍历/提取器使用）。
	Root() coreast.Node

	// SetRoot 设置语法树根节点。
	SetRoot(root coreast.Node)

	// SourceBytes 源文件字节内容（供 AST 文本切片）。
	SourceBytes() *[]byte

	// SetSourceBytes 设置源文件字节内容。
	SetSourceBytes(b *[]byte)

	// Definitions 本文件收集的定义。
	Definitions() []*DefinitionEntry

	// AddDefinition 注册一个定义并更新索引。
	AddDefinition(def *DefinitionEntry)

	// Imports 导入表：短名 -> 导入信息列表（含通配符）。
	Imports() map[string][]*ImportInfo

	// AddImport 注册一个导入。
	AddImport(shortName string, info *ImportInfo)

	// FindByShortName 按短名返回定义（可能多个）。
	FindByShortName(name string) ([]*DefinitionEntry, bool)

	// FindByQualifiedName 按限定名返回定义。
	FindByQualifiedName(qn string) (*DefinitionEntry, bool)

	// FindByKind 按元素类型返回定义。
	FindByKind(kind model.ElementKind) ([]*DefinitionEntry, bool)
}

// NewFileContext 创建一个文件上下文（返回 FileContext 接口，默认 Base 实现）。
func NewFileContext(filePath, packageName string, root coreast.Node, sourceBytes *[]byte) FileContext {
	return NewBaseFileContext(filePath, packageName, root, sourceBytes)
}

// BaseFileContext 是 FileContext 的通用实现：提供索引/去重等语言无关机制。
// 语言实现内嵌本结构复用基础能力，可按需覆盖方法。
type BaseFileContext struct {
	filePath string

	packageName string

	namespaces []string

	root coreast.Node

	definitions []*DefinitionEntry

	imports map[string][]*ImportInfo

	sourceBytes *[]byte

	// 索引：短名 -> 定义（文件内消歧）。
	shortNameMap map[string][]*DefinitionEntry

	// 索引：限定名 -> 定义。
	qualifiedNameMap map[string]*DefinitionEntry

	// 索引：元素 kind -> 定义。
	kindMap map[model.ElementKind][]*DefinitionEntry
}

// NewBaseFileContext 创建一个空的 Base 文件上下文。
func NewBaseFileContext(filePath, packageName string, root coreast.Node, sourceBytes *[]byte) *BaseFileContext {
	return &BaseFileContext{
		filePath:         filePath,
		packageName:      packageName,
		root:             root,
		imports:          make(map[string][]*ImportInfo),
		sourceBytes:      sourceBytes,
		definitions:      make([]*DefinitionEntry, 0),
		shortNameMap:     make(map[string][]*DefinitionEntry),
		qualifiedNameMap: make(map[string]*DefinitionEntry),
		kindMap:          make(map[model.ElementKind][]*DefinitionEntry),
	}
}

// FilePath 返回文件路径。
func (fc *BaseFileContext) FilePath() string { return fc.filePath }

// SetFilePath 设置文件路径。
func (fc *BaseFileContext) SetFilePath(path string) { fc.filePath = path }

// PackageName 返回包名。
func (fc *BaseFileContext) PackageName() string { return fc.packageName }

// SetPackageName 设置包名。
func (fc *BaseFileContext) SetPackageName(name string) { fc.packageName = name }

// Namespaces 返回命名空间列表。
func (fc *BaseFileContext) Namespaces() []string { return fc.namespaces }

// SetNamespaces 设置命名空间列表。
func (fc *BaseFileContext) SetNamespaces(ns []string) { fc.namespaces = ns }

// Root 返回语法树根节点。
func (fc *BaseFileContext) Root() coreast.Node { return fc.root }

// SetRoot 设置语法树根节点。
func (fc *BaseFileContext) SetRoot(root coreast.Node) { fc.root = root }

// SourceBytes 返回源文件字节内容。
func (fc *BaseFileContext) SourceBytes() *[]byte { return fc.sourceBytes }

// SetSourceBytes 设置源文件字节内容。
func (fc *BaseFileContext) SetSourceBytes(b *[]byte) { fc.sourceBytes = b }

// Definitions 返回本文件定义列表。
func (fc *BaseFileContext) Definitions() []*DefinitionEntry { return fc.definitions }

// AddDefinition 注册一个定义并更新索引。
func (fc *BaseFileContext) AddDefinition(def *DefinitionEntry) {
	fc.definitions = append(fc.definitions, def)
	fc.shortNameMap[def.Element.Name] = append(fc.shortNameMap[def.Element.Name], def)
	fc.qualifiedNameMap[def.Element.QualifiedName] = def
	fc.kindMap[def.Element.Kind] = append(fc.kindMap[def.Element.Kind], def)
}

// Imports 返回导入表。
func (fc *BaseFileContext) Imports() map[string][]*ImportInfo { return fc.imports }

// AddImport 注册一个导入。
func (fc *BaseFileContext) AddImport(shortName string, info *ImportInfo) {
	fc.imports[shortName] = append(fc.imports[shortName], info)
}

// FindByShortName 按短名返回定义（可能多个）。
func (fc *BaseFileContext) FindByShortName(name string) ([]*DefinitionEntry, bool) {
	entries, ok := fc.shortNameMap[name]
	return entries, ok
}

// FindByQualifiedName 按限定名返回定义。
func (fc *BaseFileContext) FindByQualifiedName(qn string) (*DefinitionEntry, bool) {
	entry, ok := fc.qualifiedNameMap[qn]
	return entry, ok
}

// FindByKind 按元素类型返回定义。
func (fc *BaseFileContext) FindByKind(kind model.ElementKind) ([]*DefinitionEntry, bool) {
	entries, ok := fc.kindMap[kind]
	return entries, ok
}
