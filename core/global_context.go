package core

import (
	"sync"

	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// DefinitionEntry 描述一个代码元素定义及其父级归属。
type DefinitionEntry struct {
	Element  *model.CodeElement
	ParentQN string

	// IsDeclaration 标记该条目是「声明」而非「定义」（C/C++ 前向声明、函数原型、extern 变量）。
	// 声明与定义共享同一 QN + Kind 时，索引保留定义（后到的定义原地覆盖先到的声明）。
	IsDeclaration bool

	// Node 保留该定义锚定的 AST 节点（元数据填充/作用域修正/语法糖处理用）。
	// 仅语言实现内部使用，引擎不负责清理（FileContext 随文件生命周期释放）。
	Node coreast.Node
}

// GlobalContext 描述整个分析会话的全局符号上下文。
//
// 接口化设计（2026-11-06）：core 只保留契约，通用索引/锁机制由 BaseGlobalContext 提供，
// 语言实现内嵌 Base 复用（如 C++ 用 :: 拼接 QN、按 ns 注册）。
// 注意：实现内部自带锁，调用方不得持锁再调用（避免嵌套锁死锁）。
type GlobalContext interface {
	// RegisterFileContext 注册一个文件上下文（Collector 阶段产出）。
	RegisterFileContext(fc FileContext)

	// RegisterPackage 注册包/命名空间层级（语言专属语义由实现决定）。
	// Java 拆点号包名；C++ 拆 :: 命名空间；C 空实现。
	RegisterPackage(packageName string)

	// FileContexts 返回全部文件上下文（只读快照）。
	FileContexts() map[string]FileContext

	// FileContext 按路径返回单个文件上下文。
	FileContext(relPath string) (FileContext, bool)

	// Definitions 返回全部定义（只读快照）。
	Definitions() []*DefinitionEntry

	// AddDefinition 添加一个定义（幂等：已存在的 QN+Kind 忽略）。
	AddDefinition(def *DefinitionEntry)

	// FindByQualifiedName 按限定名查找定义。
	// 声明/定义分离语言（C/C++）中同一 QN 可能存在声明+定义两个元素，
	// 此方法返回「定义优先」：优先非声明 kind（FUNCTION/METHOD），否则按注册序。
	FindByQualifiedName(qn string) (*DefinitionEntry, bool)

	// FindMethodByNoParamsQN 按不含参数的方法 QN 查找候选重载。
	FindMethodByNoParamsQN(noParamsQN string) ([]*DefinitionEntry, bool)

	// BuildQualifiedName 委托 Builder 拼接 QN。
	BuildQualifiedName(parentQN, name string) string

	// Resolver 返回语言符号解析器。
	Resolver() SymbolResolver

	// Builder 返回语言符号构建器。
	Builder() SymbolBuilder
}

// GlobalContextRules 描述语言对全局符号索引的策略差异（core 不含语言硬编码）。
type GlobalContextRules struct {
	// OverloadKinds 参与「去参 QN 重载索引」的元素 kind。
	// Java 仅 Method；C/C++ 含 Method/Function/MethodDecl/FunctionDecl。
	OverloadKinds map[model.ElementKind]struct{}
}

// BaseGlobalContext 是 GlobalContext 的通用实现：提供索引/锁/去重等语言无关机制。
// 语言实现内嵌本结构复用基础能力，可按需覆盖方法。
type BaseGlobalContext struct {
	Resolver_ SymbolResolver
	Builder_  SymbolBuilder

	rules GlobalContextRules

	mu           sync.RWMutex
	fileContexts map[string]FileContext
	definitions  []*DefinitionEntry

	// qnIndex 全量索引：key = QN + "|" + Kind。声明/定义共存（G2）。
	qnIndex map[string]*DefinitionEntry

	// qnPrimary 主索引：key = 裸 QN，指向「定义优先」的元素（O(1) 查找）。
	// 声明 kind（FUNCTION_DECL/METHOD_DECL）不覆盖 qnPrimary，故 FindByQualifiedName 定义优先。
	qnPrimary map[string]*DefinitionEntry

	methodNoParamQN map[string][]*DefinitionEntry // key: 不含方法参数的 qn
}

// NewBaseGlobalContext 创建全局上下文 Base 实现。
func NewBaseGlobalContext(resolver SymbolResolver, builder SymbolBuilder, rules GlobalContextRules) *BaseGlobalContext {
	return &BaseGlobalContext{
		Resolver_:       resolver,
		Builder_:        builder,
		rules:           rules,
		fileContexts:    make(map[string]FileContext),
		definitions:     make([]*DefinitionEntry, 0),
		qnIndex:         make(map[string]*DefinitionEntry),
		qnPrimary:       make(map[string]*DefinitionEntry),
		methodNoParamQN: make(map[string][]*DefinitionEntry),
	}
}

// RegisterFileContext 登记文件内全部定义（同一把写锁下批量完成）。
// 包/命名空间注册属于语言策略：语言上下文覆盖本方法，先调用自身 RegisterPackage 再委托本实现。
func (gc *BaseGlobalContext) RegisterFileContext(fc FileContext) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.fileContexts[fc.FilePath()] = fc
	for _, def := range fc.Definitions() {
		gc.addDefinitionLocked(def)
	}
}

// FileContexts 返回全部文件上下文（只读快照）。
func (gc *BaseGlobalContext) FileContexts() map[string]FileContext {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	out := make(map[string]FileContext, len(gc.fileContexts))
	for k, v := range gc.fileContexts {
		out[k] = v
	}
	return out
}

// FileContext 按路径返回单个文件上下文。
func (gc *BaseGlobalContext) FileContext(relPath string) (FileContext, bool) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	fc, ok := gc.fileContexts[relPath]
	return fc, ok
}

// Definitions 返回全部定义（只读快照）。
func (gc *BaseGlobalContext) Definitions() []*DefinitionEntry {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return append([]*DefinitionEntry(nil), gc.definitions...)
}

// AddDefinition 添加一个定义（幂等：已存在的 QN+Kind 忽略）。
func (gc *BaseGlobalContext) AddDefinition(def *DefinitionEntry) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.addDefinitionLocked(def)
}

// qnKey 构造全量索引键：QN + "|" + Kind。
// 声明/定义分离语言（C/C++）中声明与定义 QN 相同但 Kind 不同（FUNCTION_DECL vs FUNCTION），
// 纯 QN 去重会吞掉后注册者，故 key 含 Kind 使两者共存（G2）。
func qnKey(qn string, kind model.ElementKind) string {
	return qn + "|" + string(kind)
}

func (gc *BaseGlobalContext) addDefinitionLocked(def *DefinitionEntry) {
	key := qnKey(def.Element.QualifiedName, def.Element.Kind)
	if existing, exists := gc.qnIndex[key]; exists {
		// 声明与定义同 QN 同 Kind（C 的 struct Fwd; struct Fwd{};）：定义覆盖声明。
		if existing.IsDeclaration && !def.IsDeclaration {
			gc.replaceDefinitionLocked(existing, def)
			gc.qnIndex[key] = def
			gc.qnPrimary[def.Element.QualifiedName] = def
		}
		return
	}
	gc.definitions = append(gc.definitions, def)
	gc.qnIndex[key] = def

	// 主索引：定义优先。声明不覆盖已有定义（定义优先语义），
	// 但纯声明符号（无对应定义）仍需可查找（如 C++ 头文件成员声明），故不存在时填入；
	// 若先注册的是声明、后注册定义，则定义覆盖声明。
	if existing, exists := gc.qnPrimary[def.Element.QualifiedName]; !exists {
		gc.qnPrimary[def.Element.QualifiedName] = def
	} else if existing.IsDeclaration && !def.IsDeclaration {
		gc.qnPrimary[def.Element.QualifiedName] = def
	}

	if gc.isOverloadKind(def.Element.Kind) {
		key := gc.Builder_.BuildQualifiedName(def.ParentQN, def.Element.Name)
		gc.methodNoParamQN[key] = append(gc.methodNoParamQN[key], def)
	}
}

// isOverloadKind 判断元素 kind 是否参与重载索引（语言策略注入）。
func (gc *BaseGlobalContext) isOverloadKind(kind model.ElementKind) bool {
	_, ok := gc.rules.OverloadKinds[kind]
	return ok
}

// replaceDefinitionLocked 用定义替换已登记的声明（保持注册顺序与索引一致）。
func (gc *BaseGlobalContext) replaceDefinitionLocked(old, def *DefinitionEntry) {
	for i, d := range gc.definitions {
		if d == old {
			gc.definitions[i] = def
			break
		}
	}
	if gc.isOverloadKind(def.Element.Kind) {
		key := gc.Builder_.BuildQualifiedName(def.ParentQN, def.Element.Name)
		entries := gc.methodNoParamQN[key]
		for i, d := range entries {
			if d == old {
				entries[i] = def
				break
			}
		}
	}
}

// FindByQualifiedName 按限定名查找定义（O(1)）。
// 声明 kind 不进入 qnPrimary，故天然「定义优先」；纯声明符号（无定义）也能命中
// （声明注册时 qnPrimary 无该 QN，会填入声明本身）。
func (gc *BaseGlobalContext) FindByQualifiedName(qn string) (*DefinitionEntry, bool) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	e, ok := gc.qnPrimary[qn]
	return e, ok
}

// FindMethodByNoParamsQN 按不含参数的方法 QN 查找候选重载。
func (gc *BaseGlobalContext) FindMethodByNoParamsQN(noParamsQN string) ([]*DefinitionEntry, bool) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	entries, ok := gc.methodNoParamQN[noParamsQN]
	return entries, ok
}

// RegisterPackage 默认空实现（无包/命名空间语义的语言直接复用）。
// 语言实现内嵌 Base 后覆盖此方法即可获得包层级注册能力。
func (gc *BaseGlobalContext) RegisterPackage(packageName string) {}

// BuildQualifiedName 委托 Builder 拼接 QN。
func (gc *BaseGlobalContext) BuildQualifiedName(parentQN, name string) string {
	return gc.Builder_.BuildQualifiedName(parentQN, name)
}

// Resolver 返回语言符号解析器。
func (gc *BaseGlobalContext) Resolver() SymbolResolver { return gc.Resolver_ }

// Builder 返回语言符号构建器。
func (gc *BaseGlobalContext) Builder() SymbolBuilder { return gc.Builder_ }
