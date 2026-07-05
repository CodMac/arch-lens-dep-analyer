package java

import (
	"fmt"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/x/java/derivator"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/resolver"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/enricher/rel"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type Extractor struct{}

func NewJavaExtractor() *Extractor {
	return &Extractor{}
}

func (e *Extractor) Extract(filePath string, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gCtx.FileContexts[filePath]
	if !ok {
		return nil, fmt.Errorf("file context not found: %s", filePath)
	}

	// 1. 层级关系
	hierarchyRels := e.extractHierarchy(fCtx, gCtx)

	// 2. 结构关系
	structuralRels := e.extractStructural(fCtx, gCtx)

	// 3. 动作关系
	actionRels, err := e.discoverActionRelations(fCtx, gCtx)
	if err != nil {
		return nil, err
	}

	// 4. 元数据增强
	baseActionAndStructural := append(structuralRels, actionRels...)
	enricher := rel.NewEnricher(fCtx, gCtx)
	for _, r := range baseActionAndStructural {
		enricher.EnrichCoreMetadata(r)
	}

	// 5. 高级/派生关系
	derivator := derivator.NewRelDerivator(fCtx, gCtx)
	typeArgRels := derivator.SupplementTypeArgs(fCtx.Definitions)
	captureRels := derivator.DeriveCaptureRelations(baseActionAndStructural)

	// 6. 合并最终结果
	var allRels []*model.DependencyRelation
	allRels = append(allRels, hierarchyRels...)
	allRels = append(allRels, baseActionAndStructural...)
	allRels = append(allRels, typeArgRels...)
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
				target := gCtx.Resolver.ResolveType(gCtx, fCtx, imp.RawImportPath, imp.Kind)
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
				target := gCtx.Resolver.ResolveType(gCtx, fCtx, helper.Clean(sc), model.Class)
				rels = append(rels, &model.DependencyRelation{Type: model.Extend, Source: elem, Target: target})
			}
			if impls, ok := elem.Extra.Mores[constants.ClassImplementedInterfaces].([]string); ok {
				for _, implName := range impls {
					target := gCtx.Resolver.ResolveType(gCtx, fCtx, helper.Clean(implName), model.Interface)
					rels = append(rels, &model.DependencyRelation{Type: model.Implement, Source: elem, Target: target})
				}
			}
		}
		if elem.Kind == model.AnonymousClass {
			if ac, ok := elem.Extra.Mores[constants.AnonymousClassType].(string); ok && ac != "" {
				target := gCtx.Resolver.ResolveType(gCtx, fCtx, helper.Clean(ac), model.Class)
				rels = append(rels, &model.DependencyRelation{Type: model.Extend, Source: elem, Target: target})
			}
		}

		// --- 2. 处理 Interface (Extend) ---
		if elem.Kind == model.Interface {
			if impls, ok := elem.Extra.Mores[constants.InterfaceImplementedInterfaces].([]string); ok {
				for _, implName := range impls {
					target := gCtx.Resolver.ResolveType(gCtx, fCtx, helper.Clean(implName), model.Interface)
					rels = append(rels, &model.DependencyRelation{Type: model.Extend, Source: elem, Target: target})
				}
			}
		}

		// --- 3. 处理注解 (Annotation) ---
		for _, anno := range elem.Extra.Annotations {
			target := gCtx.Resolver.ResolveType(gCtx, fCtx, helper.Clean(anno), model.KAnnotation)
			rels = append(rels, &model.DependencyRelation{
				Type: model.Annotation, Source: elem, Target: target,
				Mores: map[string]interface{}{constants.RelRawText: anno},
			})
		}

		// --- 4. 处理方法签名 (Parameter/Return/Throw) ---
		if elem.Kind == model.Method {
			// Parameter
			if pts, ok := elem.Extra.Mores[constants.MethodParameters].([]string); ok {
				for _, p := range pts {
					typePart := e.extractTypeFromParam(p)
					target := gCtx.Resolver.ResolveType(gCtx, fCtx, helper.Clean(typePart), model.Class)
					rels = append(rels, &model.DependencyRelation{
						Type: model.Parameter, Source: elem, Target: target,
						Mores: map[string]interface{}{constants.RelRawText: p},
					})
				}
			}

			// Return
			if rt, ok := elem.Extra.Mores[constants.MethodReturnType].(string); ok && rt != "void" && rt != "" {
				target := gCtx.Resolver.ResolveType(gCtx, fCtx, helper.Clean(rt), model.Class)
				rels = append(rels, &model.DependencyRelation{
					Type: model.Return, Source: elem, Target: target,
					Mores: map[string]interface{}{constants.RelRawText: rt},
				})
			}

			// Throw
			if ths, ok := elem.Extra.Mores[constants.MethodThrowsTypes].([]string); ok {
				for _, ex := range ths {
					target := gCtx.Resolver.ResolveType(gCtx, fCtx, helper.Clean(ex), model.Class)
					rels = append(rels, &model.DependencyRelation{
						Type: model.Throw, Source: elem, Target: target,
						Mores: map[string]interface{}{constants.RelRawText: ex},
					})
				}
			}
		}
	}
	return rels
}

func (e *Extractor) discoverActionRelations(fCtx *core.FileContext, gCtx *core.GlobalContext) ([]*model.DependencyRelation, error) {
	// 1. 第一阶段：一次性全遍历，收集洗干净的捕获点集合
	captures, err := e.getCaptures(fCtx)
	if err != nil {
		return nil, err
	}

	// 2. 第二阶段：基于返回的捕获点集合，集中进行处理分发和生成 Rel
	ncResolver := resolver.NewNodeContextResolver(fCtx)
	var allRels []*model.DependencyRelation

	for _, ct := range captures {
		sourceElem := e.determinePreciseSource(ct.Node, fCtx)
		if sourceElem == nil {
			continue
		}

		actionTargets := e.mapAction(ct.CapName, ct.Node, fCtx, gCtx)
		for _, at := range actionTargets {
			if at.RelType == "" || at.Target == nil {
				continue
			}

			// Use 关系二次业务过滤
			if at.RelType == model.Use && !e.isUseRel(ct.Node, at.Target) {
				continue
			}

			// 提取上下文并调用工厂装配
			ncResult := ncResolver.ResolveContext(at.RelType, ct.Node)
			rel := e.buildRelation(fCtx, ct, at, ncResult)
			allRels = append(allRels, rel)
		}
	}

	return allRels, nil
}

// =============================================================================
// 辅助函数
// =============================================================================

type CaptureTarget struct {
	CapName string
	Node    *sitter.Node
}

func (e *Extractor) getCaptures(fCtx *core.FileContext) ([]*CaptureTarget, error) {
	tsLang, _ := core.GetLanguage(core.LangJava)
	q, err := sitter.NewQuery(tsLang, JavaActionQuery)
	if err != nil {
		return nil, err
	}
	defer q.Close()

	qc := sitter.NewQueryCursor()
	matches := qc.Matches(q, fCtx.RootNode, *fCtx.SourceBytes)

	captures := make([]*CaptureTarget, 0)
	useCaps := make([]*CaptureTarget, 0)
	useFilterList := make(map[uintptr]bool)

	// 全量扫描，遍历匹配项
	for {
		match := matches.Next()
		if match == nil {
			break
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]
			if !strings.HasSuffix(capName, "_target") && capName != "explicit_constructor_stmt" && capName != "id_atom" {
				continue
			}

			// 通过局部变量重新分配内存，规避指针复用隐患
			nodeCopy := cap.Node
			target := &CaptureTarget{CapName: capName, Node: &nodeCopy}

			if capName == "id_atom" {
				useCaps = append(useCaps, target)
			} else {
				useFilterList[cap.Node.Id()] = true
				captures = append(captures, target)
			}
		}
	}

	// useCaps过滤
	for _, uc := range useCaps {
		// 第一重清洗：拦截冲突节点
		if useFilterList[uc.Node.Id()] {
			continue
		}
		// 第二重清洗：拦截非合法的静态读取点噪声（声明、定义等）
		if !e.isStaticUseNode(uc.Node) {
			continue
		}

		captures = append(captures, uc)
	}

	return captures, nil
}

func (e *Extractor) determinePreciseSource(n *sitter.Node, fCtx *core.FileContext) *model.CodeElement {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {
		line := int(curr.StartPosition().Row) + 1
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

type actionTarget struct {
	RelType    model.DependencyType
	TargetNode *sitter.Node
	Target     *model.CodeElement
}

func (e *Extractor) mapAction(capName string, node *sitter.Node, fCtx *core.FileContext, gCtx *core.GlobalContext) []actionTarget {
	switch capName {
	case "call_target", "ref_target":
		return []actionTarget{
			{model.Call, node, gCtx.Resolver.ResolveAction(gCtx, fCtx, node, model.Call)},
		}
	case "create_target", "explicit_constructor_stmt":
		return []actionTarget{
			{model.Create, node, gCtx.Resolver.ResolveAction(gCtx, fCtx, node, model.Create)},
			{model.Call, node, gCtx.Resolver.ResolveAction(gCtx, fCtx, node, model.Call)},
		}
	case "cast_target":
		return []actionTarget{
			{model.Cast, node, gCtx.Resolver.ResolveAction(gCtx, fCtx, node, model.Cast)},
		}
	case "assign_target":
		return []actionTarget{
			{model.Assign, node, gCtx.Resolver.ResolveAction(gCtx, fCtx, node, model.Assign)},
		}
	case "id_atom":
		return []actionTarget{
			{model.Use, node, gCtx.Resolver.ResolveAction(gCtx, fCtx, node, model.Use)},
		}
	case "throw_target":
		return []actionTarget{
			{model.Throw, node, gCtx.Resolver.ResolveAction(gCtx, fCtx, node, model.Throw)},
		}
	default:
		return nil
	}
}

func (e *Extractor) isStaticUseNode(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	parent := node.Parent()
	if parent == nil {
		return false
	}

	if nameNode := parent.ChildByFieldName("name"); nameNode != nil && nameNode.Id() == node.Id() {
		return false
	}

	switch parent.Kind() {
	case "variable_declarator", "formal_parameter",
		"class_declaration", "interface_declaration", "enum_declaration",
		"method_declaration", "constructor_declaration", "package_declaration", "import_declaration",
		"type_parameter", "labeled_statement":
		return false

	case "assignment_expression":
		if leftNode := parent.ChildByFieldName("left"); leftNode != nil && leftNode.Id() == node.Id() {
			return false
		}

	case "type_identifier":
		return false

	case "method_invocation", "method_reference":
		if nameNode := parent.ChildByFieldName("name"); nameNode != nil && nameNode.Id() == node.Id() {
			return false
		}
	}

	return true
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

func (e *Extractor) buildRelation(fCtx *core.FileContext, ct *CaptureTarget, at actionTarget, ncResult *resolver.Result) *model.DependencyRelation {
	sourceElem := e.determinePreciseSource(ct.Node, fCtx)
	return &model.DependencyRelation{
		Type:     at.RelType,
		Source:   sourceElem,
		Target:   at.Target,
		Location: helper.ExtractLocation(at.TargetNode, fCtx.FilePath),
		Mores: map[string]interface{}{
			constants.TmpNode:           at.TargetNode,
			constants.TmpExpressNode:    ncResult.ExpressNode,
			constants.TmpCtxNode:        ncResult.ContextNode,
			constants.RelNodeAstKind:    ct.Node.Kind(),
			constants.RelExpressAstKind: ncResult.ExpressNode.Kind(),
			constants.RelContextAstKind: ncResult.ContextNode.Kind(),
			constants.RelRawText:        ncResult.ContextNode.Utf8Text(*fCtx.SourceBytes),
		},
	}
}

// =============================================================================
// export函数
// =============================================================================

func (e *Extractor) GetCaptures(fCtx *core.FileContext) ([]*CaptureTarget, error) {
	return e.getCaptures(fCtx)
}
