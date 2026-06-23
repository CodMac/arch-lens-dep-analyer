package core

import (
	"fmt"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// --- 语言特有的符号解析接口 ---

type SymbolResolver interface {
	// BuildQualifiedName 根据父节点和当前名构建 QN
	// (Java 用 ".", C++ 用 "::")
	BuildQualifiedName(parentQN, name string) string

	// RegisterPackage 注册包/命名空间逻辑
	// (Java 需要拆分点号，Go 只需要单层)
	RegisterPackage(gc *GlobalContext, packageName string)

	// IsPrimitive 是否为基础类型
	IsPrimitive(t string) bool

	// ResolveType 解析结构体符号(Package、Class、Interface、AnonymousClass、Enum......), 如果上下文没找到，则返回kind类型的外部实体
	ResolveType(gc *GlobalContext, fc *FileContext, symbol string, kind model.ElementKind) *model.CodeElement

	// ResolveFunc 解析方法符号(Method、MethodRef)
	ResolveFunc(gc *GlobalContext, fc *FileContext, node *sitter.Node, receiver, symbol string) *model.CodeElement

	// ResolveVar 解析变量符号(Field、Variable)
	ResolveVar(gc *GlobalContext, fc *FileContext, node *sitter.Node, receiver, symbol string) *model.CodeElement
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
