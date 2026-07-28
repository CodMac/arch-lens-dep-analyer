package java

import (
	"strings"

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
// 对象缓存池管理
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
// 核心接口实现
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
	symbol := strings.TrimSpace(node.Utf8Text(*fCtx.SourceBytes))

	// 1. 三段式第一步：一元化 Context 提取
	ncResolver := jsr.getNcResolver(fCtx)
	contextResult := ncResolver.ResolveContext(relType, node)
	if contextResult == nil || contextResult.ExpressNode == nil {
		return jsr.createExternalFallback(fCtx, symbol, jsr.getFallbackKindByRelType(relType))
	}

	// 2. 三段式第二步：将物理表达树解析为符号分段
	segmenter := jsr.getExpressionSegmenter(fCtx)
	chain := segmenter.Segment(contextResult.ExpressNode, relType)
	if chain == nil {
		return jsr.createExternalFallback(fCtx, symbol, jsr.getFallbackKindByRelType(relType))
	}

	// 3. 三段式第三步：已知符号链路推导求值
	chainResolver := jsr.getChainResolver(gCtx, fCtx)
	targetEle := chainResolver.ResolveChain(chain)

	// 4. 判定推导符号与当前依赖动作的合拍性 (Alignment)
	if alignedEle := jsr.alignElementWithRelType(gCtx, targetEle, relType); alignedEle != nil {
		return alignedEle
	}

	// 5. 无法推导或推导失败：根据上下文链条生成最优雅的虚拟外部节点（全局保底）
	return jsr.generateContextualFallback(fCtx, chain, relType)
}

// =============================================================================
// 依赖动作合拍性校正 (Action Alignment)
// =============================================================================

// alignElementWithRelType 校验推导出的 CodeElement 是否符合依赖动作的预期，必要时进行语义修正或二次回溯
func (jsr *SymbolResolver) alignElementWithRelType(gCtx *core.GlobalContext, targetEle *model.CodeElement, relType model.DependencyType) *model.CodeElement {
	if targetEle == nil {
		return nil
	}

	switch relType {
	case model.Call:
		if targetEle.Kind == model.Method {
			return targetEle
		}
		// 如果推导出的结果是类/接口，但动作是 CALL，说明可能是未注册具体构造方法的实例化过程
		if targetEle.Kind == model.Class || targetEle.Kind == model.Interface {
			return jsr.createExternalFallbackWithParent(targetEle.QualifiedName, targetEle.Name, model.Method)
		}

	case model.Use, model.Assign:
		if targetEle.Kind == model.Field || targetEle.Kind == model.Variable {
			return targetEle
		}

	default:
		// 对于 CREATE, CAST, THROW, RETURN 等动作，期望返回 Class/Interface/AnonymousClass 等宏观类型
		if targetEle.Kind == model.Class || targetEle.Kind == model.Interface || targetEle.Kind == model.AnonymousClass {
			return targetEle
		}
		// 若意外定位到了变量或方法，则回溯其声明所属的 Owner Class
		if ownerClass := helper.GetOwnerClass(gCtx, targetEle); ownerClass != nil {
			return ownerClass
		}
	}

	return nil
}

// =============================================================================
// 高阶全局兜底逻辑
// =============================================================================

// generateContextualFallback 依据拉平后的分段链，拼装出最吻合的外部 QualifiedName
func (jsr *SymbolResolver) generateContextualFallback(fCtx *core.FileContext, chain *resolver.ExpressionChain, relType model.DependencyType) *model.CodeElement {
	if chain == nil {
		return jsr.createExternalFallback(fCtx, "Unknown", jsr.getFallbackKindByRelType(relType))
	}

	// 1. 精准根据 Head 的语法类型提取起点 HeadName（对于 Cast 关系，此处拿到的就是 CastType，如 "String" 或 "List"）
	headName := jsr.extractHeadName(chain.Head)
	if idx := strings.Index(headName, "<"); idx != -1 {
		headName = headName[:idx]
	}

	baseQN := headName
	if imps, ok := fCtx.Imports[headName]; ok && len(imps) > 0 {
		baseQN = imps[0].RawImportPath
	}

	// 2. 如果带有后续 Segments（例如 ((String) obj).getBytes()），累加后续 Segments 路径
	var builder strings.Builder
	builder.WriteString(baseQN)

	for _, seg := range chain.Segments {
		if seg.Name != "" {
			if seg.Kind == resolver.SegmentClass {
				builder.WriteString("$")
			} else {
				builder.WriteString(".")
			}
			builder.WriteString(seg.Name)
		}
	}

	fallbackQN := builder.String()
	fallbackKind := jsr.getFallbackKindByRelType(relType)

	// 计算 Short Name
	shortName := fallbackQN
	if idx := strings.LastIndexAny(fallbackQN, ".$"); idx != -1 {
		shortName = fallbackQN[idx+1:]
	}

	// 特殊动作类型微调
	if relType == model.Call {
		if !strings.HasSuffix(shortName, "()") {
			shortName += "()"
		}
		if !strings.HasSuffix(fallbackQN, "()") {
			fallbackQN += "()"
		}
	}

	return &model.CodeElement{
		Name:           shortName,
		QualifiedName:  fallbackQN,
		Kind:           fallbackKind,
		IsFormExternal: true,
	}
}

// extractHeadName 根据 Head 类型解析出准确的 Base 符号名
func (jsr *SymbolResolver) extractHeadName(head resolver.ExpressionHead) string {
	switch head.Type {
	case resolver.HeadCastExpr:
		if head.CastType != "" {
			return head.CastType
		}
		return head.Name
	case resolver.HeadNewExpr:
		return head.Name
	default:
		return head.Name
	}
}

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

func (jsr *SymbolResolver) createExternalFallback(fCtx *core.FileContext, symbolName string, defaultKind model.ElementKind) *model.CodeElement {
	qualifiedName := symbolName
	if idx := strings.Index(symbolName, "<"); idx != -1 {
		symbolName = symbolName[:idx]
		qualifiedName = symbolName
	}

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

func (jsr *SymbolResolver) createExternalFallbackWithParent(parentQN, shortName string, kind model.ElementKind) *model.CodeElement {
	targetQN := parentQN + "." + shortName
	showName := shortName
	if kind == model.Method {
		targetQN += "()"
		showName += "()"
	}
	return &model.CodeElement{
		Name:           showName,
		QualifiedName:  targetQN,
		Kind:           kind,
		IsFormExternal: true,
	}
}
