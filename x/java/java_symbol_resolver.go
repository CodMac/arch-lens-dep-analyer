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
	if _, ok := jsr.ncResolverMap[fCtx.FilePath]; !ok {
		jsr.ncResolverMap[fCtx.FilePath] = resolver.NewNodeContextResolver(fCtx)
	}

	return jsr.ncResolverMap[fCtx.FilePath]
}

func (jsr *SymbolResolver) getExpressionSegmenter(fCtx *core.FileContext) *resolver.ExpressionSegmenter {
	if _, ok := jsr.segmenterMap[fCtx.FilePath]; !ok {
		jsr.segmenterMap[fCtx.FilePath] = resolver.NewExpressionSegmenter(fCtx)
	}

	return jsr.segmenterMap[fCtx.FilePath]
}

func (jsr *SymbolResolver) getChainResolver(gCtx *core.GlobalContext, fCtx *core.FileContext) *resolver.ChainResolver {
	if _, ok := jsr.chainResolverMap[fCtx.FilePath]; !ok {
		jsr.chainResolverMap[fCtx.FilePath] = resolver.NewChainResolver(gCtx, fCtx)
	}

	return jsr.chainResolverMap[fCtx.FilePath]
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

	return jsr.createExternalFallback(fCtx, symbol, kind)
}

func (jsr *SymbolResolver) ResolveAction(gCtx *core.GlobalContext, fCtx *core.FileContext, node *sitter.Node, relType model.DependencyType) *model.CodeElement {
	if node == nil {
		return nil
	}
	symbol := node.Utf8Text(*fCtx.SourceBytes)

	// 调用统一的流水线总入口 ResolveContext，注入依赖关系获取双轨解析容器
	ncResolver := jsr.getNcResolver(fCtx)
	contextResult := ncResolver.ResolveContext(relType, node)
	if contextResult == nil || contextResult.ExpressNode == nil {
		return jsr.createExternalFallback(fCtx, symbol, model.Class)
	}

	// 将拉平后的表达链输送给平铺切片处理器
	segmenter := jsr.getExpressionSegmenter(fCtx)
	chain := segmenter.Segment(contextResult.ExpressNode)
	if chain == nil {
		return jsr.createExternalFallback(fCtx, symbol, model.Class)
	}

	// 解析
	chainResolver := jsr.getChainResolver(gCtx, fCtx)
	result := chainResolver.ResolveChain(chain)

	if result != nil {
		return result
	}

	return jsr.createExternalFallback(fCtx, symbol, model.Class)
}

// =============================================================================
// 辅助函数
// =============================================================================

func (jsr *SymbolResolver) createExternalFallback(fCtx *core.FileContext, symbolName string, defaultKind model.ElementKind) *model.CodeElement {
	qualifiedName := symbolName

	// 如果当前 fallback 的短名存在于 Imports 映射中，自动升级为全限定名
	if imps, ok := fCtx.Imports[symbolName]; ok && len(imps) > 0 {
		qualifiedName = imps[0].RawImportPath
	}

	return &model.CodeElement{
		Name:           symbolName,
		QualifiedName:  qualifiedName,
		Kind:           defaultKind,
		IsFormExternal: true,
	}
}
