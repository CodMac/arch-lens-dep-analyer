package core

import (
	"sync"

	"github.com/CodMac/arch-lens-dep-analyer/model"
)

type GlobalContext struct {
	FileContexts map[string]*FileContext
	Definitions  []*DefinitionEntry
	Resolver     SymbolResolver // 持有具体语言的解析器
	Builder      SymbolBuilder  // 持有具体语言的构建器
	mutex        sync.RWMutex

	qualifiedNameMap      map[string]*DefinitionEntry
	methodMapWithNoParams map[string][]*DefinitionEntry // key: 不包含方法参数的qn
}

func NewGlobalContext(resolver SymbolResolver, builder SymbolBuilder) *GlobalContext {
	return &GlobalContext{
		FileContexts: make(map[string]*FileContext),
		Definitions:  make([]*DefinitionEntry, 0),
		Resolver:     resolver,
		Builder:      builder,

		qualifiedNameMap:      make(map[string]*DefinitionEntry),
		methodMapWithNoParams: make(map[string][]*DefinitionEntry),
	}
}

// RegisterFileContext 逻辑现在调用 Resolver 处理包名
func (gc *GlobalContext) RegisterFileContext(fc *FileContext) {
	gc.mutex.Lock()
	defer gc.mutex.Unlock()

	gc.FileContexts[fc.FilePath] = fc

	// 1. 委托 Resolver 处理包/命名空间注册 (Java 拆分, Go 不拆)
	gc.Resolver.RegisterPackage(gc, fc.PackageName)

	// 2. 注册文件内定义 (AddDefinition 会自动忽略已存在的 QN，即忽略重复的 FILE 节点)
	for _, entry := range fc.Definitions {
		gc.AddDefinition(entry)
	}
}

func (gc *GlobalContext) AddDefinition(def *DefinitionEntry) {
	defQN := def.Element.QualifiedName

	_, ok := gc.qualifiedNameMap[defQN]
	if !ok {
		gc.Definitions = append(gc.Definitions, def)
		gc.qualifiedNameMap[defQN] = def

		if def.Element.Kind == model.Method {
			methodKey := gc.Builder.BuildQualifiedName(def.ParentQN, def.Element.Name)
			gc.methodMapWithNoParams[methodKey] = append(gc.methodMapWithNoParams[methodKey], def)
		}
	}
}

func (gc *GlobalContext) FindByQualifiedName(qn string) (*DefinitionEntry, bool) {
	entry, ok := gc.qualifiedNameMap[qn]
	return entry, ok
}

func (gc *GlobalContext) BuildQualifiedName(parentQN, name string) string {
	return gc.Builder.BuildQualifiedName(parentQN, name)
}

func (gc *GlobalContext) FindMethodByNoParamsQN(noParamsQN string) ([]*DefinitionEntry, bool) {
	entries, ok := gc.methodMapWithNoParams[noParamsQN]
	return entries, ok
}

func (gc *GlobalContext) RLock() { gc.mutex.RLock() }

func (gc *GlobalContext) RUnlock() { gc.mutex.RUnlock() }
