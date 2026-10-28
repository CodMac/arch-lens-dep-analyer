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

	// Node 保留该定义锚定的 AST 节点（元数据填充/作用域修正/语法糖处理用）。
	// 仅语言实现内部使用，引擎不负责清理（FileContext 随文件生命周期释放）。
	Node coreast.Node
}

// GlobalContext 描述整个分析会话的全局符号上下文。
// 统一由内部 RWMutex 保护：所有公开方法自行加锁，调用方不得持有锁再调用（避免嵌套锁死锁）。
type GlobalContext struct {
	Resolver SymbolResolver
	Builder  SymbolBuilder

	mu              sync.RWMutex
	fileContexts    map[string]*FileContext
	definitions     []*DefinitionEntry
	qnIndex         map[string]*DefinitionEntry
	methodNoParamQN map[string][]*DefinitionEntry // key: 不含方法参数的 qn
}

// NewGlobalContext 创建全局上下文。
func NewGlobalContext(resolver SymbolResolver, builder SymbolBuilder) *GlobalContext {
	return &GlobalContext{
		Resolver:        resolver,
		Builder:         builder,
		fileContexts:    make(map[string]*FileContext),
		definitions:     make([]*DefinitionEntry, 0),
		qnIndex:         make(map[string]*DefinitionEntry),
		methodNoParamQN: make(map[string][]*DefinitionEntry),
	}
}

// RegisterFileContext 注册一个文件上下文（Collector 阶段产出）。
// 先委托 Resolver 注册包名（Resolver 内部自行加锁，不能在本函数持锁时调用），
// 再登记文件内全部定义（同一把写锁下批量完成）。
func (gc *GlobalContext) RegisterFileContext(fc *FileContext) {
	gc.Resolver.RegisterPackage(gc, fc.PackageName)

	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.fileContexts[fc.FilePath] = fc
	for _, def := range fc.Definitions {
		gc.addDefinitionLocked(def)
	}
}

// FileContexts 返回全部文件上下文（只读快照）。
func (gc *GlobalContext) FileContexts() map[string]*FileContext {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	out := make(map[string]*FileContext, len(gc.fileContexts))
	for k, v := range gc.fileContexts {
		out[k] = v
	}
	return out
}

// FileContext 按路径返回单个文件上下文。
func (gc *GlobalContext) FileContext(relPath string) (*FileContext, bool) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	fc, ok := gc.fileContexts[relPath]
	return fc, ok
}

// Definitions 返回全部定义（只读快照）。
func (gc *GlobalContext) Definitions() []*DefinitionEntry {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	return append([]*DefinitionEntry(nil), gc.definitions...)
}

// AddDefinition 添加一个定义（幂等：已存在的 QN 忽略）。
func (gc *GlobalContext) AddDefinition(def *DefinitionEntry) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	gc.addDefinitionLocked(def)
}

func (gc *GlobalContext) addDefinitionLocked(def *DefinitionEntry) {
	if _, exists := gc.qnIndex[def.Element.QualifiedName]; exists {
		return
	}
	gc.definitions = append(gc.definitions, def)
	gc.qnIndex[def.Element.QualifiedName] = def

	if def.Element.Kind == model.Method {
		key := gc.Builder.BuildQualifiedName(def.ParentQN, def.Element.Name)
		gc.methodNoParamQN[key] = append(gc.methodNoParamQN[key], def)
	}
}

// FindByQualifiedName 按限定名查找定义。
func (gc *GlobalContext) FindByQualifiedName(qn string) (*DefinitionEntry, bool) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	e, ok := gc.qnIndex[qn]
	return e, ok
}

// FindMethodByNoParamsQN 按不含参数的方法 QN 查找候选重载。
func (gc *GlobalContext) FindMethodByNoParamsQN(noParamsQN string) ([]*DefinitionEntry, bool) {
	gc.mu.RLock()
	defer gc.mu.RUnlock()
	entries, ok := gc.methodNoParamQN[noParamsQN]
	return entries, ok
}

// BuildQualifiedName 委托 Builder 拼接 QN。
func (gc *GlobalContext) BuildQualifiedName(parentQN, name string) string {
	return gc.Builder.BuildQualifiedName(parentQN, name)
}
