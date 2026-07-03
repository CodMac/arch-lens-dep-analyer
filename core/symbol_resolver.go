package core

import (
	"fmt"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// --- 语言特有的符号解析接口 ---

type SymbolResolver interface {
	// RegisterPackage 注册包/命名空间逻辑
	// (Java 需要拆分点号，Go 只需要单层)
	RegisterPackage(gc *GlobalContext, packageName string)

	// ResolveType 解析结构体符号(Package、Class、Interface、AnonymousClass、Enum......)
	// 如果上下文没找到，则返回kind类型的外部实体
	ResolveType(gc *GlobalContext, fc *FileContext, symbol string, kind model.ElementKind) *model.CodeElement

	// ResolveAction 统一的动作解析入口，根据目标节点、上下文节点和关系类型进行解析
	ResolveAction(gc *GlobalContext, fc *FileContext, node *sitter.Node, relType model.DependencyType) *model.CodeElement
}

var symbolResolverMap = make(map[Language]SymbolResolver)

// RegisterSymbolResolver 注册一个语言与其对应的 SymbolResolver 工厂函数。
func RegisterSymbolResolver(lang Language, resolver SymbolResolver) {
	symbolResolverMap[lang] = resolver
}

// GetSymbolResolver 根据语言类型获取对应的 SymbolResolver 实例。
func GetSymbolResolver(lang Language) (SymbolResolver, error) {
	resolver, ok := symbolResolverMap[lang]
	if !ok {
		return nil, fmt.Errorf("no SymbolResolver for language: %s", lang)
	}

	return resolver, nil
}
