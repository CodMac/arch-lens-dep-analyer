package java

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/resolver"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type SymbolResolver struct {
	ncResolverMap    map[string]*resolver.NodeContextResolver
	segmenterMap     map[string]*resolver.ExpressionSegmenter
	chainResolverMap map[string]*resolver.ChainResolver
}

func NewSymbolResolver() *SymbolResolver {
	return &SymbolResolver{
		ncResolverMap:    make(map[string]*resolver.NodeContextResolver),
		segmenterMap:     make(map[string]*resolver.ExpressionSegmenter),
		chainResolverMap: make(map[string]*resolver.ChainResolver),
	}
}

// =============================================================================
// 对象Map
// =============================================================================

func (jsr *SymbolResolver) getNcResolver(fCtx *core.FileContext) *resolver.NodeContextResolver {
	contextResolver := jsr.ncResolverMap[fCtx.FilePath]
	if contextResolver == nil {
		contextResolver = resolver.NewNodeContextResolver(fCtx)
	}

	return contextResolver
}

func (jsr *SymbolResolver) getExpressionSegmenter(fCtx *core.FileContext) *resolver.ExpressionSegmenter {
	segmenter := jsr.segmenterMap[fCtx.FilePath]
	if segmenter == nil {
		segmenter = resolver.NewExpressionSegmenter(fCtx)
	}

	return segmenter
}

func (jsr *SymbolResolver) getChainResolver(gCtx *core.GlobalContext, fCtx *core.FileContext) *resolver.ChainResolver {
	chainResolver := jsr.chainResolverMap[fCtx.FilePath]
	if chainResolver == nil {
		chainResolver = resolver.NewChainResolver(gCtx, fCtx)
	}

	return chainResolver
}

// =============================================================================
// 接口实现
// =============================================================================

func (jsr *SymbolResolver) RegisterPackage(gCtx *core.GlobalContext, packageName string) {
	if packageName == "" {
		return
	}
	pkgElement := &model.CodeElement{
		Name:          packageName,
		QualifiedName: packageName,
		Kind:          model.Package,
	}
	gCtx.AddDefinition(&core.DefinitionEntry{
		Element:  pkgElement,
		ParentQN: "",
	})
}

func (jsr *SymbolResolver) ResolveType(gCtx *core.GlobalContext, fCtx *core.FileContext, symbol string, kind model.ElementKind) *model.CodeElement {
	entries := helper.PreciseResolve(gCtx, fCtx, symbol)
	if len(entries) > 0 {
		return entries[0].Element
	}
	return &model.CodeElement{
		Name:           symbol,
		QualifiedName:  symbol,
		Kind:           kind,
		IsFormExternal: true,
	}
}

func (jsr *SymbolResolver) ResolveAction(gCtx *core.GlobalContext, fCtx *core.FileContext, node *sitter.Node, relType model.DependencyType) *model.CodeElement {
	if node == nil {
		return nil
	}

	// 调用统一的流水线总入口 ResolveContext，注入依赖关系获取双轨解析容器
	ncResolver := jsr.getNcResolver(fCtx)
	contextResult := ncResolver.ResolveContext(relType, node)
	if contextResult == nil || contextResult.ExpressNode == nil {
		return jsr.createExternalFallback(node.Utf8Text(*fCtx.SourceBytes))
	}

	// 将拉平后的表达链输送给平铺切片处理器
	segmenter := jsr.getExpressionSegmenter(fCtx)
	chain := segmenter.Segment(contextResult.ExpressNode)
	if chain == nil {
		return jsr.createExternalFallback(node.Utf8Text(*fCtx.SourceBytes))
	}

	// 解析
	chainResolver := jsr.getChainResolver(gCtx, fCtx)
	result := chainResolver.ResolveChain(chain)

	if result != nil {
		return result
	}

	return jsr.createExternalFallback(node.Utf8Text(*fCtx.SourceBytes))
}

// =============================================================================
// 辅助函数
// =============================================================================

func (jsr *SymbolResolver) createExternalFallback(symbolName string) *model.CodeElement {
	return &model.CodeElement{
		Name:           symbolName,
		QualifiedName:  symbolName,
		Kind:           model.Class,
		IsFormExternal: true,
	}
}
