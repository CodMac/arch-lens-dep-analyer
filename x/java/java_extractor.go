package java

import (
	"fmt"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/x/java/resolver"

	"github.com/CodMac/arch-lens-dep-analyer/x/java/enricher/rel"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Extractor struct {
	resolver core.SymbolResolver
}

func NewJavaExtractor() *Extractor {
	return &Extractor{
		resolver: NewJavaSymbolResolver(),
	}
}

// =============================================================================
// 主流水线 (Main Pipeline)
// =============================================================================

func (e *Extractor) Extract(filePath string, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gCtx.FileContexts[filePath]
	if !ok {
		return nil, fmt.Errorf("file context not found: %s", filePath)
	}

	// 1. 静态结构 + 动作发现
	hierarchyRels := e.extractHierarchy(fCtx, gCtx)
	structuralRels := e.extractStructural(fCtx, gCtx)
	actionRels, err := e.discoverActionRelations(fCtx, gCtx)
	if err != nil {
		return nil, err
	}

	// 2. 元数据增强
	enhanceTargets := append(structuralRels, actionRels...)
	e.enrichRelMetadata(enhanceTargets, fCtx, gCtx)

	// 3. 链式语法处理（基于已提取到的CALL关系）

	// 4. Capture关系发现（基于已提取到的ASSIGN和USE关系）
	captureRels := e.genCaptureRelations(enhanceTargets)

	// 5. 合并结果
	var allRels []*model.DependencyRelation
	allRels = append(allRels, hierarchyRels...)
	allRels = append(allRels, structuralRels...)
	allRels = append(allRels, actionRels...)
	allRels = append(allRels, captureRels...)

	return allRels, nil
}

func (e *Extractor) extractHierarchy(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation

	fileDef, ok := gCtx.FindByQualifiedName(fCtx.FilePath)
	if ok {
		fileSource := fileDef.Element
		for _, imports := range fCtx.Imports {
			for _, imp := range imports {
				target := e.resolver.ResolveType(gCtx, fCtx, imp.RawImportPath, imp.Kind)
				rels = append(rels, &model.DependencyRelation{
					Type: model.Import, Source: fileSource, Target: target, Location: imp.Location,
				})
			}
		}
	}

	for _, entry := range fCtx.Definitions {
		if entry.ParentQN != "" {
			if parent, ok := gCtx.FindByQualifiedName(entry.ParentQN); ok {
				rels = append(rels, &model.DependencyRelation{Type: model.Contain, Source: parent.Element, Target: entry.Element})
			}
		}
	}
	return rels
}

func (e *Extractor) extractStructural(fCtx *core.FileContext, gCtx *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, entry := range fCtx.Definitions {
		elem := entry.Element
		if elem.Extra == nil {
			continue
		}

		// --- 1. 处理 Class (Extend/Implement) ---
		if elem.Kind == model.Class {
			if sc, ok := elem.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
				target := e.resolver.ResolveType(gCtx, fCtx, helper.Clean(sc), model.Class)
				rels = append(rels, &model.DependencyRelation{Type: model.Extend, Source: elem, Target: target})
			}
			if impls, ok := elem.Extra.Mores[constants.ClassImplementedInterfaces].([]string); ok {
				for _, implName := range impls {
					target := e.resolver.ResolveType(gCtx, fCtx, helper.Clean(implName), model.Interface)
					rels = append(rels, &model.DependencyRelation{Type: model.Implement, Source: elem, Target: target})
				}
			}
		}
		if elem.Kind == model.AnonymousClass {
			if ac, ok := elem.Extra.Mores[constants.AnonymousClassType].(string); ok && ac != "" {
				target := e.resolver.ResolveType(gCtx, fCtx, helper.Clean(ac), model.Class)
				rels = append(rels, &model.DependencyRelation{Type: model.Extend, Source: elem, Target: target})
			}
		}

		// --- 2. 处理 Interface (Extend) ---
		if elem.Kind == model.Interface {
			if impls, ok := elem.Extra.Mores[constants.InterfaceImplementedInterfaces].([]string); ok {
				for _, implName := range impls {
					target := e.resolver.ResolveType(gCtx, fCtx, helper.Clean(implName), model.Interface)
					rels = append(rels, &model.DependencyRelation{Type: model.Extend, Source: elem, Target: target})
				}
			}
		}

		// --- 3. 处理注解 (Annotation) ---
		for _, anno := range elem.Extra.Annotations {
			target := e.resolver.ResolveType(gCtx, fCtx, helper.Clean(anno), model.KAnnotation)
			rels = append(rels, &model.DependencyRelation{
				Type: model.Annotation, Source: elem, Target: target,
				Mores: map[string]interface{}{constants.RelRawText: anno},
			})
		}

		// --- 4. 处理方法签名 (Parameter/Return/Throw) ---
		if elem.Kind == model.Method {
			if pts, ok := elem.Extra.Mores[constants.MethodParameters].([]string); ok {
				for _, p := range pts {
					typePart := e.extractTypeFromParam(p)
					target := e.resolver.ResolveType(gCtx, fCtx, helper.Clean(typePart), model.Class)
					rels = append(rels, &model.DependencyRelation{
						Type: model.Parameter, Source: elem, Target: target,
						Mores: map[string]interface{}{constants.RelRawText: p},
					})
				}
			}
			if rt, ok := elem.Extra.Mores[constants.MethodReturnType].(string); ok && rt != "void" && rt != "" {
				target := e.resolver.ResolveType(gCtx, fCtx, helper.Clean(rt), model.Class)
				rels = append(rels, &model.DependencyRelation{
					Type: model.Return, Source: elem, Target: target,
					Mores: map[string]interface{}{constants.RelRawText: rt},
				})
			}
			if ths, ok := elem.Extra.Mores[constants.MethodThrowsTypes].([]string); ok {
				for _, ex := range ths {
					target := e.resolver.ResolveType(gCtx, fCtx, helper.Clean(ex), model.Class)
					rels = append(rels, &model.DependencyRelation{
						Type: model.Throw, Source: elem, Target: target,
						Mores: map[string]interface{}{constants.RelRawText: ex},
					})
				}
			}
		}

		// --- 5. 处理变量泛型 (TypeArg) ---
		for _, rt := range e.getRawTypesForTypeArgs(elem) {
			rels = append(rels, e.collectAllTypeArgs(rt, elem, gCtx, fCtx)...)
		}

	}
	return rels
}

func (e *Extractor) discoverActionRelations(fCtx *core.FileContext, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	tsLang, _ := core.GetLanguage(core.LangJava)
	q, err := sitter.NewQuery(tsLang, JavaActionQuery)
	if err != nil {
		return nil, err
	}
	defer q.Close()

	var rels []*model.DependencyRelation
	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, fCtx.RootNode, *fCtx.SourceBytes)

	for {
		match := matches.Next()
		if match == nil {
			break
		}
		capturedNode := &match.Captures[0].Node
		sourceElem := e.determinePreciseSource(capturedNode, fCtx, gCtx)
		if sourceElem == nil {
			continue
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]
			if !strings.HasSuffix(capName, "_target") && capName != "explicit_constructor_stmt" && capName != "id_atom" {
				continue
			}

			// 1. 调用 mapAction 获取动作定义
			actionTargets := e.mapAction(capName, &cap.Node, fCtx, gCtx)
			for _, at := range actionTargets {
				if at.RelType == "" || at.Target == nil {
					continue
				}

				rels = append(rels, &model.DependencyRelation{
					Type:     at.RelType,
					Source:   sourceElem,
					Target:   at.Target, // 使用 mapAction resolve 好的对象
					Location: helper.ExtractLocation(at.TargetNode, fCtx.FilePath),
					Mores: map[string]interface{}{
						constants.RelRawText: resolver.GetRawTextForAction(string(at.RelType), at.TargetNode, at.ContextNode, fCtx.SourceBytes),
						constants.TmpNode:    at.TargetNode,
						constants.TmpCtxNode: at.ContextNode,
					},
				})
			}
		}
	}
	return rels, nil
}

func (e *Extractor) enrichRelMetadata(enhanceTargets []*model.DependencyRelation, fCtx *core.FileContext, gCtx *core.GlobalContext) {
	enricher := rel.NewEnricher(e.resolver, fCtx, gCtx)

	for _, rel := range enhanceTargets {
		enricher.EnrichCoreMetadata(rel)
	}
}

func (e *Extractor) processChainRels(actionRels []*model.DependencyRelation, fCtx *core.FileContext, gCtx *core.GlobalContext) {
	chainedCallResolver := resolver.NewChainedCallResolver(e.resolver, gCtx, fCtx)

	for _, rel := range actionRels {
		if rel.Type != model.Call || rel.Target == nil {
			continue
		}

		chainedCallResolver.ProcessChainRels(rel)
	}

}

func (e *Extractor) genCaptureRelations(deps []*model.DependencyRelation) []*model.DependencyRelation {
	var captures []*model.DependencyRelation
	seen := make(map[string]bool)

	for _, rel := range deps {
		if rel.Source == nil || rel.Target == nil {
			continue
		}

		isCapture := false

		if rel.Type == model.Use {
			if val, ok := rel.Mores[constants.RelUseIsCapture]; ok {
				if b, isBool := val.(bool); isBool && b {
					isCapture = true
				}
			}
		}

		if rel.Type == model.Assign {
			if val, ok := rel.Mores[constants.RelAssignIsCapture]; ok {
				if b, isBool := val.(bool); isBool && b {
					isCapture = true
				}
			}
		}

		if isCapture {
			key := rel.Source.QualifiedName + "->" + rel.Target.QualifiedName

			if !seen[key] {
				seen[key] = true
				captureRel := &model.DependencyRelation{
					Source:   rel.Source,
					Target:   rel.Target,
					Type:     model.Capture,
					Location: rel.Location,
					Mores:    make(map[string]interface{}),
				}
				captures = append(captures, captureRel)
			}
		}
	}
	return captures
}

// =============================================================================
// 辅助函数
// =============================================================================

func (e *Extractor) determinePreciseSource(n *sitter.Node, fCtx *core.FileContext, gCtx *core.GlobalContext) *model.CodeElement {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {

		// 行号
		line := int(curr.StartPosition().Row) + 1

		// 类型
		var k model.ElementKind
		switch curr.Kind() {
		case "method_declaration", "constructor_declaration":
			k = model.Method
		case "static_initializer":
			k = model.ScopeBlock
		case "lambda_expression":
			k = model.Lambda
		case "field_declaration":
			k = model.Field
		case "variable_declarator":
			if p := curr.Parent(); p != nil && p.Kind() == "field_declaration" {
				k = model.Field
			} else {
				continue
			}
		case "class_body", "interface_body", "program":
			return nil
		default:
			continue
		}

		// 根据行号+类型，确定父容器
		if defs, ok := fCtx.FindByElementKind(k); ok {
			for _, entry := range defs {
				if entry.Element.Kind == k && entry.Element.Location.StartLine == line {
					return entry.Element
				}
			}
		}
	}
	return nil
}

type ActionTarget struct {
	RelType     model.DependencyType
	TargetNode  *sitter.Node
	ContextNode *sitter.Node
	Target      *model.CodeElement
}

func (e *Extractor) mapAction(capName string, node *sitter.Node, fCtx *core.FileContext, gCtx *core.GlobalContext) []ActionTarget {
	// 创建上下文解析器
	ctxResolver := resolver.NewNodeContextResolver()

	switch capName {
	case "call_target", "ref_target":
		// CALL保持原逻辑，因为它已经正确
		ctxNode := helper.FindNearestKind(node, "method_invocation", "method_reference", "explicit_constructor_invocation", "object_creation_expression")
		if ctxNode == nil {
			return nil
		}
		return []ActionTarget{{RelType: model.Call, TargetNode: node, ContextNode: ctxNode, Target: e.resolver.ResolveAction(gCtx, fCtx, node, ctxNode, model.Call)}}

	case "create_target":
		ctxNode := helper.FindNearestKind(node, "object_creation_expression", "array_creation_expression")
		if ctxNode == nil {
			return nil
		}
		return []ActionTarget{
			{model.Create, node, ctxNode, e.resolver.ResolveAction(gCtx, fCtx, node, ctxNode, model.Create)},
			{model.Call, node, ctxNode, e.resolver.ResolveAction(gCtx, fCtx, node, ctxNode, model.Call)},
		}

	case "cast_target":
		ctxNode := helper.FindNearestKind(node, "cast_expression", "instanceof_expression")
		if ctxNode == nil {
			return nil
		}
		return []ActionTarget{{model.Cast, node, ctxNode, e.resolver.ResolveAction(gCtx, fCtx, node, ctxNode, model.Cast)}}

	case "assign_target":
		// 使用新的上下文解析器
		result := ctxResolver.ResolveContext("ASSIGN", node)
		if result == nil || result.ContextNode == nil {
			return nil
		}
		return []ActionTarget{{model.Assign, node, result.ContextNode, e.resolver.ResolveAction(gCtx, fCtx, node, result.ContextNode, model.Assign)}}

	case "id_atom":
		// 使用新的上下文解析器
		result := ctxResolver.ResolveContext("USE", node)
		if result == nil || result.ContextNode == nil {
			return nil
		}

		target := e.resolver.ResolveAction(gCtx, fCtx, node, result.ContextNode, model.Use)
		if !e.isUseRel(node, target) {
			return nil
		}
		return []ActionTarget{{model.Use, node, result.ContextNode, target}}

	case "throw_target":
		ctxNode := helper.FindNearestKind(node, "throw_statement")
		if ctxNode == nil {
			return nil
		}
		return []ActionTarget{{model.Throw, node, ctxNode, e.resolver.ResolveAction(gCtx, fCtx, node, ctxNode, model.Throw)}}

	case "explicit_constructor_stmt":
		ctxNode := node
		if ctxNode == nil {
			return nil
		}
		return []ActionTarget{
			{model.Call, node, ctxNode, e.resolver.ResolveAction(gCtx, fCtx, node, ctxNode, model.Call)},
			{model.Create, node, ctxNode, e.resolver.ResolveAction(gCtx, fCtx, node, ctxNode, model.Create)},
		}

	default:
		return nil
	}
}

func (e *Extractor) isUseRel(node *sitter.Node, target *model.CodeElement) bool {
	// 1. 快速定位符号：如果不是变量或字段，直接 pass
	if target == nil || (target.Kind != model.Variable && target.Kind != model.Field) {
		return false
	}

	// 2. 排除定义点：通过父节点的 FieldName 判断
	// 如果当前 identifier 是其父节点的 "name" 字段，说明它是声明，不是使用
	parent := node.Parent()
	if parent.ChildByFieldName("name") != nil && parent.ChildByFieldName("name").Id() == node.Id() {
		return false
	}

	// 3. 排除特定语法噪声
	switch parent.Kind() {
	case "method_invocation", "method_reference":
		// 如果是方法调用/引用的名称部分，由 call_target 处理
		if nameNode := parent.ChildByFieldName("name"); nameNode != nil && nameNode.Id() == node.Id() {
			return false
		}
	case "scoped_identifier", "package_declaration", "import_declaration":
		return false
	}

	return true
}

func (e *Extractor) extractTypeFromParam(p string) string {
	parts := strings.Fields(p)
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return p
}

func (e *Extractor) getRawTypesForTypeArgs(elem *model.CodeElement) (res []string) {
	keys := []string{constants.FieldRawType, constants.VariableRawType, constants.MethodReturnType}

	for _, k := range keys {
		if v, ok := elem.Extra.Mores[k].(string); ok {
			res = append(res, v)
		}
	}

	if pts, ok := elem.Extra.Mores[constants.MethodParameters].([]string); ok {
		for _, p := range pts {
			res = append(res, e.extractTypeFromParam(p))
		}
	}

	return
}

func (e *Extractor) parseTypeArgs(rawType string) []string {
	start, end := strings.Index(rawType, "<"), strings.LastIndex(rawType, ">")
	if start == -1 || end == -1 || start >= end {
		return nil
	}

	content := rawType[start+1 : end]

	var args []string
	bracketLevel := 0
	current := strings.Builder{}
	for _, r := range content {
		switch r {
		case '<':
			bracketLevel++
			current.WriteRune(r)
		case '>':
			bracketLevel--
			current.WriteRune(r)
		case ',':
			if bracketLevel == 0 {
				args = append(args, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, strings.TrimSpace(current.String()))
	}

	return args
}

func (e *Extractor) collectAllTypeArgs(rt string, source *model.CodeElement, gCtx *core.GlobalContext, fCtx *core.FileContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation

	if !strings.Contains(rt, "<") {
		return nil
	}

	args := e.parseTypeArgs(rt)
	for i, arg := range args {
		target := e.resolver.ResolveType(gCtx, fCtx, helper.Clean(arg), model.Class)
		rels = append(rels, &model.DependencyRelation{
			Type: model.TypeArg, Source: source, Target: target,
			Mores: map[string]interface{}{constants.RelTypeArgIndex: i, constants.RelRawText: arg, constants.RelAstKind: "type_arguments"},
		})

		if strings.Contains(arg, "<") {
			rels = append(rels, e.collectAllTypeArgs(arg, source, gCtx, fCtx)...)
		}
	}
	return rels
}
