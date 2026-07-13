package java

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
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

	// 1. 三段式流程：第一段 context 解析
	ncResolver := jsr.getNcResolver(fCtx)
	contextResult := ncResolver.ResolveContext(relType, node)
	if contextResult == nil || contextResult.ExpressNode == nil {
		return jsr.createExternalFallback(fCtx, symbol, jsr.getFallbackKindByRelType(relType))
	}

	// 2. 三段式流程：第二段 表达式分段
	segmenter := jsr.getExpressionSegmenter(fCtx)
	chain := segmenter.Segment(contextResult.ExpressNode)
	if chain == nil {
		return jsr.createExternalFallback(fCtx, symbol, jsr.getFallbackKindByRelType(relType))
	}

	// 3. 三段式流程：第三段 链式推导
	chainResolver := jsr.getChainResolver(gCtx, fCtx)
	targetEle := chainResolver.ResolveChain(chain)
	if targetEle == nil {
		return jsr.createExternalFallback(fCtx, symbol, jsr.getFallbackKindByRelType(relType))
	}

	// 基于依赖关系 relType 智能对齐返回的符号类型
	switch relType {
	case model.Call:
		// CALL 期望返回方法
		if targetEle.Kind == model.Method {
			return targetEle
		}

	case model.Use, model.Assign:
		// USE、ASSIGN 期望返回变量或字段
		if targetEle.Kind == model.Field || targetEle.Kind == model.Variable {
			return targetEle
		}

	default:
		// 对于 RETURN、THROW、CAST、CREATE 等其他类型，期望返回 Class/Type 实体
		if targetEle.Kind != model.Class && targetEle.Kind != model.Interface && targetEle.Kind != model.AnonymousClass {
			class := helper.GetOwnerClass(gCtx, targetEle)
			if class != nil {
				return class
			}
		}
	}

	return jsr.createExternalFallback(fCtx, symbol, jsr.getFallbackKindByRelType(relType))
}

// =============================================================================
// 辅助函数
// =============================================================================

func (jsr *SymbolResolver) getFallbackKindByRelType(relType model.DependencyType) model.ElementKind {
	switch relType {
	case model.Call:
		return model.Method
	case model.Use, model.Assign:
		return model.Field
	default:
		return model.Class
	}
}

func (jsr *SymbolResolver) extractMethodReturnType(gCtx *core.GlobalContext, fCtx *core.FileContext, methodElement *model.CodeElement) *model.CodeElement {
	if methodElement == nil || methodElement.Extra == nil {
		return nil
	}
	var returnQN string
	if qn, ok := methodElement.Extra.Mores[constants.MethodReturnTypeWithQN].(string); ok && qn != "" {
		returnQN = qn
	} else if raw, ok := methodElement.Extra.Mores[constants.MethodReturnType].(string); ok {
		returnQN = raw
	}

	if idx := strings.Index(returnQN, "<"); idx != -1 {
		returnQN = strings.TrimSpace(returnQN[:idx])
	}

	if returnQN != "" && returnQN != "void" {
		if entries := helper.PreciseResolve(gCtx, fCtx, returnQN); len(entries) > 0 {
			return entries[0].Element
		}

		// 外部 fallback
		shortName := returnQN
		if idx := strings.LastIndex(returnQN, "."); idx != -1 {
			shortName = returnQN[idx+1:]
		}
		return &model.CodeElement{
			Name:           shortName,
			QualifiedName:  returnQN,
			Kind:           model.Class,
			IsFormExternal: true,
		}
	}
	return nil
}

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
