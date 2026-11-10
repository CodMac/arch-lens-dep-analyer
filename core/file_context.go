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

	// Location 导入声明位置。
	Location *model.Location

	// Attrs 语言专属属性（core 不解释其语义）。
	// 约定 key 使用语言前缀，避免跨语言冲突：
	//   - Java:  "java.import.is_wildcard" (bool)
	//   - C/C++: "cpp.include.is_system"   (bool)
	Attrs map[string]any
}

// GetAttr 读取语言专属属性；不存在时返回 nil。
func (i *ImportInfo) GetAttr(key string) any {
	if i == nil || i.Attrs == nil {
		return nil
	}
	return i.Attrs[key]
}

// SetAttr 写入语言专属属性（Attrs 为 nil 时自动初始化）。
func (i *ImportInfo) SetAttr(key string, value any) {
	if i.Attrs == nil {
		i.Attrs = make(map[string]any)
	}
	i.Attrs[key] = value
}

// BoolAttr 读取布尔属性；不存在或类型不符时返回 false。
func (i *ImportInfo) BoolAttr(key string) bool {
	v, _ := i.GetAttr(key).(bool)
	return v
}

// FileContext 描述单个源文件的符号定义集合与索引。
//
// 分层设计：core 只保留语言无关契约，通用索引/去重机制由 BaseFileContext 提供；
// 语言专属能力通过可选能力接口暴露（如 Java 的 PackageAwareFileContext），
// core 接口本身不含任何语言概念。
type FileContext interface {
	// FilePath 文件路径（相对项目根）。
	FilePath() string

	// SetFilePath 设置文件路径（引擎路径相对化时调用）。
	SetFilePath(path string)

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

	// Imports 导入表：短名 -> 导入信息列表（同名可多条）。
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

// PackageAwareFileContext 是语言专属的可选能力接口：
// 单包语义的语言（Java/Go）实现该接口，其余语言无需实现。
type PackageAwareFileContext interface {
	FileContext

	// PackageName 包名（如 com.example.base）。
	PackageName() string

	// SetPackageName 设置包名。
	SetPackageName(name string)
}

// BaseFileContext 是 FileContext 的通用实现：提供索引/去重等语言无关机制。
// 语言实现内嵌本结构复用基础能力，可按需覆盖方法。
type BaseFileContext struct {
	filePath string

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

	// 索引：QN + "|" + Kind -> definitions 下标（声明/定义同键原地替换用）。
	declDefIndex map[string]int
}

// NewBaseFileContext 创建一个空的 Base 文件上下文。
func NewBaseFileContext(filePath string, root coreast.Node, sourceBytes *[]byte) *BaseFileContext {
	return &BaseFileContext{
		filePath:         filePath,
		root:             root,
		imports:          make(map[string][]*ImportInfo),
		sourceBytes:      sourceBytes,
		definitions:      make([]*DefinitionEntry, 0),
		shortNameMap:     make(map[string][]*DefinitionEntry),
		qualifiedNameMap: make(map[string]*DefinitionEntry),
		kindMap:          make(map[model.ElementKind][]*DefinitionEntry),
		declDefIndex:     make(map[string]int),
	}
}

// FilePath 返回文件路径。
func (fc *BaseFileContext) FilePath() string { return fc.filePath }

// SetFilePath 设置文件路径。
func (fc *BaseFileContext) SetFilePath(path string) { fc.filePath = path }

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
// 同 QN + Kind 的「声明 → 定义」原地替换，避免同文件前向声明遮蔽后续定义。
func (fc *BaseFileContext) AddDefinition(def *DefinitionEntry) {
	key := def.Element.QualifiedName + "|" + string(def.Element.Kind)
	if idx, ok := fc.declDefIndex[key]; ok {
		old := fc.definitions[idx]
		if old.IsDeclaration && !def.IsDeclaration {
			fc.definitions[idx] = def
			if list := fc.shortNameMap[old.Element.Name]; list != nil {
				for i, d := range list {
					if d == old {
						list[i] = def
						break
					}
				}
			}
			if list := fc.kindMap[def.Element.Kind]; list != nil {
				for i, d := range list {
					if d == old {
						list[i] = def
						break
					}
				}
			}
			fc.qualifiedNameMap[def.Element.QualifiedName] = def
		}
		return
	}
	if fc.declDefIndex == nil {
		fc.declDefIndex = make(map[string]int)
	}
	fc.declDefIndex[key] = len(fc.definitions)
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
