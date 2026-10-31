// Package extractor 实现 Java 语言 core.Extractor 接口：
// 从单个文件提取静态结构关系、动作关系（Call/Create/Assign/Use/Cast/Throw）
// 并基于动作关系推导 Capture 关系，最后统一做元数据增强。
package extractor

import (
	"fmt"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	javast "github.com/CodMac/arch-lens-dep-analyer/x/java/ast"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/enricher/rel"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/extractor/derived"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/query"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/symbol_resolver"
	sitter "github.com/tree-sitter/go-tree-sitter"
	treeSitterJava "github.com/tree-sitter/tree-sitter-java/bindings/go"
)

// Extractor 实现 core.Extractor。
type Extractor struct {
	resolver core.SymbolResolver
}

// New 创建 Java Extractor。
func New() *Extractor {
	return &Extractor{
		resolver: symbol_resolver.New(),
	}
}

// =============================================================================
// 主流水线 (Main Pipeline)
// =============================================================================

// Extract 提取指定文件（相对路径）的全部依赖关系。
func (e *Extractor) Extract(relPath string, gc *core.GlobalContext) ([]*model.DependencyRelation, error) {
	fCtx, ok := gc.FileContext(relPath)
	if !ok {
		return nil, fmt.Errorf("file context not found: %s", relPath)
	}

	// 1. 静态结构 + 动作发现
	hierarchyRels := e.extractHierarchy(fCtx, gc)
	structuralRels := e.extractStructural(fCtx, gc)
	actionRels, err := e.discoverActionRelations(fCtx, gc)
	if err != nil {
		return nil, err
	}

	// 2. 元数据增强（写入 is_capture 标记）
	enhanceTargets := append(structuralRels, actionRels...)
	e.enrichRelMetadata(enhanceTargets, fCtx, gc)

	// 3. 派生关系（TypeArg + Capture）
	derivedRels := derived.ExtractAll(&derived.Context{
		GC: gc, FC: fCtx, Resolver: e.resolver, Deps: enhanceTargets,
	})

	// 4. 合并结果
	var allRels []*model.DependencyRelation
	allRels = append(allRels, hierarchyRels...)
	allRels = append(allRels, structuralRels...)
	allRels = append(allRels, actionRels...)
	allRels = append(allRels, derivedRels...)

	return allRels, nil
}

// extractHierarchy 提取 Import 与 Contain 关系。
func (e *Extractor) extractHierarchy(fCtx *core.FileContext, gc *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation

	fileDef, ok := gc.FindByQualifiedName(fCtx.FilePath)
	if ok {
		fileSource := fileDef.Element
		for _, imports := range fCtx.Imports {
			for _, imp := range imports {
				target := e.resolver.ResolveType(gc, fCtx, imp.RawImportPath, imp.Kind)
				rels = append(rels, &model.DependencyRelation{
					Type: model.Import, Source: fileSource, Target: target, Location: imp.Location,
				})
			}
		}
	}

	for _, entry := range fCtx.Definitions {
		if entry.ParentQN != "" {
			if parent, ok := gc.FindByQualifiedName(entry.ParentQN); ok {
				rels = append(rels, &model.DependencyRelation{Type: model.Contain, Source: parent.Element, Target: entry.Element})
			}
		}
	}
	return rels
}

// extractStructural 提取 Extend/Implement/Annotation/Parameter/Return/Throw/TypeArg。
func (e *Extractor) extractStructural(fCtx *core.FileContext, gc *core.GlobalContext) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, entry := range fCtx.Definitions {
		elem := entry.Element
		if elem.Extra == nil {
			continue
		}

		// --- 1. 处理 Class (Extend/Implement) ---
		if elem.Kind == model.Class {
			if sc, ok := elem.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
				target := e.resolver.ResolveType(gc, fCtx, helper.Clean(sc), model.Class)
				if target != nil {
					rels = append(rels, &model.DependencyRelation{Type: model.Extend, Source: elem, Target: target})
				}
			}
			if impls, ok := elem.Extra.Mores[constants.ClassImplementedInterfaces].([]string); ok {
				for _, implName := range impls {
					target := e.resolver.ResolveType(gc, fCtx, helper.Clean(implName), model.Interface)
					if target != nil {
						rels = append(rels, &model.DependencyRelation{Type: model.Implement, Source: elem, Target: target})
					}
				}
			}
		}
		if elem.Kind == model.AnonymousClass {
			if ac, ok := elem.Extra.Mores[constants.AnonymousClassType].(string); ok && ac != "" {
				target := e.resolver.ResolveType(gc, fCtx, helper.Clean(ac), model.Class)
				if target == nil {
					continue
				}
				// 匿名类目标类型：内部 Class -> 继承（EXTEND）；接口或外部类型 -> 实现（IMPLEMENT）。
				// 外部类型（未 import，如 Runnable/Comparable）PreciseResolve 未命中，
				// ResolveType 兜底返回外部 Class（IsFormExternal=true），语义上属实现接口。
				retype := model.Extend
				if target.IsFormExternal || target.Kind != model.Class {
					retype = model.Implement
				}
				rels = append(rels, &model.DependencyRelation{Type: retype, Source: elem, Target: target})
			}
		}

		// --- 2. 处理 Interface (Extend) ---
		if elem.Kind == model.Interface {
			if impls, ok := elem.Extra.Mores[constants.InterfaceImplementedInterfaces].([]string); ok {
				for _, implName := range impls {
					target := e.resolver.ResolveType(gc, fCtx, helper.Clean(implName), model.Interface)
					if target != nil {
						rels = append(rels, &model.DependencyRelation{Type: model.Extend, Source: elem, Target: target})
					}
				}
			}
		}

		// --- 3. 处理注解 (Annotation) ---
		for _, anno := range elem.Extra.Annotations {
			target := e.resolver.ResolveType(gc, fCtx, helper.Clean(anno), model.KAnnotation)
			if target != nil {
				rels = append(rels, &model.DependencyRelation{
					Type: model.Annotation, Source: elem, Target: target,
					Mores: map[string]interface{}{constants.RelRawText: anno},
				})
			}
		}

		// --- 4. 处理方法签名 (Parameter/Return/Throw) ---
		if elem.Kind == model.Method {
			if pts, ok := elem.Extra.Mores[constants.MethodParameters].([]string); ok {
				for _, p := range pts {
					typePart := e.extractTypeFromParam(p)
					target := e.resolver.ResolveType(gc, fCtx, helper.Clean(typePart), model.Class)
					// 泛型变量参数（T data）解析为 nil，不产关系。
					if target == nil {
						continue
					}
					rels = append(rels, &model.DependencyRelation{
						Type: model.Parameter, Source: elem, Target: target,
						Mores: map[string]interface{}{constants.RelRawText: p},
					})
				}
			}
			if rt, ok := elem.Extra.Mores[constants.MethodReturnType].(string); ok && rt != "void" && rt != "" {
				target := e.resolveTypeForElement(gc, fCtx, elem, helper.Clean(rt))
				if target == nil {
					goto afterReturn
				}
				rels = append(rels, &model.DependencyRelation{
					Type: model.Return, Source: elem, Target: target,
					Mores: map[string]interface{}{constants.RelRawText: rt},
				})
			}
		afterReturn:
			// 构造器隐式返回自身类型：为构造器补一条指向宿主类的 RETURN。
			// 构造器无返回类型字段（MethodReturnType 缺失），但 Java 语义上构造调用即产出该类实例。
			if isCtor, ok := elem.Extra.Mores[constants.MethodIsConstructor].(bool); ok && isCtor {
				if host, found := gc.FindByQualifiedName(entry.ParentQN); found {
					rels = append(rels, &model.DependencyRelation{
						Type: model.Return, Source: elem, Target: host.Element,
						Mores: map[string]interface{}{constants.RelRawText: entry.ParentQN},
					})
				}
			}
			if ths, ok := elem.Extra.Mores[constants.MethodThrowsTypes].([]string); ok {
				for _, ex := range ths {
					target := e.resolver.ResolveType(gc, fCtx, helper.Clean(ex), model.Class)
					if target == nil {
						continue
					}
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

// discoverActionRelations 执行动作查询，映射为动作关系。
func (e *Extractor) discoverActionRelations(fCtx *core.FileContext, gc *core.GlobalContext) ([]*model.DependencyRelation, error) {
	lang := sitter.NewLanguage(treeSitterJava.Language())
	q, qErr := sitter.NewQuery(lang, query.JavaActionQuery)
	if qErr != nil {
		return nil, fmt.Errorf("compile java action query: %s", qErr.Message)
	}
	defer q.Close()

	var rels []*model.DependencyRelation
	qc := sitter.NewQueryCursor()
	root := javast.Unwrap(fCtx.Root)
	matches := qc.Matches(q, root, *fCtx.SourceBytes)

	for {
		match := matches.Next()
		if match == nil {
			break
		}

		// 首个捕获用于确定 Source（@xxx_stmt 或首个 target）。
		if len(match.Captures) == 0 {
			continue
		}
		firstNode := javast.Wrap(&match.Captures[0].Node)
		sourceElem := e.determinePreciseSource(firstNode, fCtx, gc)
		if sourceElem == nil {
			continue
		}

		for _, cap := range match.Captures {
			capName := q.CaptureNames()[cap.Index]
			if !strings.HasSuffix(capName, "_target") && capName != "explicit_constructor_stmt" && capName != "id_atom" && capName != "field_atom" && capName != "class_lit_atom" {
				continue
			}

			capNode := javast.Wrap(&cap.Node)

			// 1. 调用 mapAction 获取动作定义
			actionTargets := e.mapAction(capName, capNode, fCtx, gc)
			for _, at := range actionTargets {
				if at.RelType == "" || at.Target == nil {
					continue
				}

				relObj := &model.DependencyRelation{
					Type:     at.RelType,
					Source:   sourceElem,
					Target:   at.Target, // 使用 mapAction resolve 好的对象
					Location: helper.ExtractLocation(at.TargetNode, fCtx.FilePath),
				}
				// 私有临时字段传递 AST 节点（enricher 阶段消费，流水线末尾清理）。
				relObj.SetTmpNode(at.TargetNode)
				relObj.SetTmpCtx(at.ContextNode)
				rels = append(rels, relObj)
			}
		}
	}
	return rels, nil
}

// enrichRelMetadata 对动作/结构关系做类型级元数据增强。
func (e *Extractor) enrichRelMetadata(enhanceTargets []*model.DependencyRelation, fCtx *core.FileContext, gc *core.GlobalContext) {
	enricher := rel.NewEnricher(e.resolver, fCtx, gc)

	for _, relObj := range enhanceTargets {
		enricher.EnrichCoreMetadata(relObj)
	}
}

// =============================================================================
// 辅助函数
// =============================================================================

// determinePreciseSource 向上定位关系发起者（方法/块/Lambda/字段/枚举常量/类型级注解宿主）。
func (e *Extractor) determinePreciseSource(n coreast.Node, fCtx *core.FileContext, gc *core.GlobalContext) *model.CodeElement {
	for curr := n.Parent(); curr != nil; curr = curr.Parent() {

		// 行号
		line := int(curr.StartPoint().Row) + 1

		// 类型
		var k model.ElementKind
		switch curr.Kind() {
		case "method_declaration", "constructor_declaration", "annotation_type_element_declaration":
			k = model.Method
		case "static_initializer":
			k = model.ScopeBlock
		case "block":
			// 实例初始化块（父为 class_body）归为 ScopeBlock；
			// 普通块（方法体/循环体等）继续向上定位到所属方法/静态块/Lambda。
			if p := curr.Parent(); p != nil && p.Kind() == "class_body" {
				k = model.ScopeBlock
			} else {
				continue
			}
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
		case "enum_constant":
			// 枚举常量：枚举构造器实参（DefectType.java: DATA_CLASS(..., InfusionList.class, ...)）
			// 归属到枚举常量元素，而非其所在枚举。
			k = model.EnumConstant
		case "modifiers":
			// 类型级注解（@MappedTypes/@FeignClient/@SpringBootApplication 等）：
			// class_literal 祖先链为 annotation -> modifiers -> 类型声明（不经 class_body）。
			// 归到所属类型元素（Class/Interface/Enum/注解类型）。
			td := curr.Parent()
			if td == nil {
				continue
			}
			switch td.Kind() {
			case "class_declaration", "record_declaration":
				k = model.Class
			case "interface_declaration":
				k = model.Interface
			case "enum_declaration":
				k = model.Enum
			case "annotation_type_declaration":
				k = model.KAnnotation
			default:
				continue
			}
			line = int(td.StartPoint().Row) + 1
		case "class_body", "interface_body", "program":
			return nil
		default:
			continue
		}

		// 根据行号+类型，确定父容器
		if defs, ok := fCtx.FindByKind(k); ok {
			for _, entry := range defs {
				if entry.Element.Kind == k && entry.Element.Location.StartLine == line {
					return entry.Element
				}
			}
		}
	}
	return nil
}

// ActionTarget 描述 mapAction 产出的一个动作目标。
type ActionTarget struct {
	RelType     model.DependencyType
	TargetNode  coreast.Node
	ContextNode coreast.Node
	Target      *model.CodeElement
}

// mapAction 依据捕获名将动作节点映射为关系目标。
func (e *Extractor) mapAction(capName string, node coreast.Node, fCtx *core.FileContext, gc *core.GlobalContext) []ActionTarget {
	var ctxNode coreast.Node

	switch capName {
	case "call_target", "ref_target":
		ctxNode = helper.FindNearestKind(node, "method_invocation", "method_reference", "explicit_constructor_invocation", "object_creation_expression")
		if ctxNode == nil {
			return nil
		}
		return []ActionTarget{{RelType: model.Call, TargetNode: node, ContextNode: ctxNode, Target: e.resolver.ResolveAction(gc, fCtx, node, model.Call)}}

	case "create_target":
		ctxNode = helper.FindNearestKind(node, "object_creation_expression", "array_creation_expression")
		if ctxNode == nil {
			return nil
		}
		return []ActionTarget{
			{RelType: model.Create, TargetNode: node, ContextNode: ctxNode, Target: e.resolver.ResolveAction(gc, fCtx, node, model.Create)},
			{RelType: model.Call, TargetNode: node, ContextNode: ctxNode, Target: e.resolver.ResolveAction(gc, fCtx, node, model.Call)},
		}

	case "cast_target":
		ctxNode = helper.FindNearestKind(node, "cast_expression", "instanceof_expression")
		if ctxNode == nil {
			return nil
		}
		return []ActionTarget{{RelType: model.Cast, TargetNode: node, ContextNode: ctxNode, Target: e.resolver.ResolveAction(gc, fCtx, node, model.Cast)}}

	case "assign_target":
		ctxNode = helper.FindNearestKind(node, "assignment_expression", "variable_declarator", "update_expression")
		if ctxNode == nil {
			return nil
		}
		return []ActionTarget{{RelType: model.Assign, TargetNode: node, ContextNode: ctxNode, Target: e.resolver.ResolveAction(gc, fCtx, node, model.Assign)}}

	case "id_atom":
		ctxNode = helper.FindNearestKind(node, "expression_statement", "local_variable_declaration", "enhanced_for_statement", "binary_expression", "cast_expression", "array_access", "parenthesized_expression", "field_access", "lambda_expression", "assignment_expression", "return_statement")
		if ctxNode == nil {
			return nil
		}
		target := e.resolver.ResolveAction(gc, fCtx, node, model.Use)
		if !e.isUseRel(node, target) {
			return nil
		}
		return []ActionTarget{{RelType: model.Use, TargetNode: node, ContextNode: ctxNode, Target: target}}

	case "field_atom":
		// 字段访问的字段名：枚举常量/外部静态字段访问（HttpStatus.NOT_FOUND）。
		// 该 identifier 是 field_access 的 field，NodeContextResolver 会爬升到 field_access 整体，
		// 由 ChainResolver 解析为 Field/EnumConstant（内部定义或外部符号）。
		ctxNode = helper.FindNearestKind(node, "expression_statement", "field_access", "binary_expression", "parenthesized_expression", "return_statement", "arguments")
		if ctxNode == nil {
			return nil
		}
		target := e.resolver.ResolveAction(gc, fCtx, node, model.Use)
		// 放行：内部枚举常量（EnumConstant）或外部字段/枚举常量（isUseRel 只收内部 Variable/Field）。
		if target != nil && target.Kind == model.EnumConstant && !target.IsFormExternal {
			return []ActionTarget{{RelType: model.Use, TargetNode: node, ContextNode: ctxNode, Target: target}}
		}
		if target != nil && target.IsFormExternal && (target.Kind == model.Field || target.Kind == model.EnumConstant) {
			return []ActionTarget{{RelType: model.Use, TargetNode: node, ContextNode: ctxNode, Target: target}}
		}
		return nil

	case "class_lit_atom":
		// 类字面量：Xxx.class 是类型引用，目标为类型元素（内部或外部）。
		// 覆盖场景：局部变量/字段初始化、方法调用实参（argument_list）、
		// 注解参数（element_value_pair/annotation_argument_list/element_value_array_initializer）。
		ctxNode = helper.FindNearestKind(node, "expression_statement", "local_variable_declaration", "method_invocation", "argument_list", "element_value_pair", "annotation_argument_list", "element_value_array_initializer", "annotation_type_element_declaration", "binary_expression", "return_statement", "field_access")
		if ctxNode == nil {
			return nil
		}
		target := e.resolver.ResolveAction(gc, fCtx, node, model.Use)
		if target != nil && (target.Kind == model.Class || target.Kind == model.Interface || target.Kind == model.Enum || target.Kind == model.KAnnotation) {
			return []ActionTarget{{RelType: model.Use, TargetNode: node, ContextNode: ctxNode, Target: target}}
		}
		return nil

	case "throw_target":
		ctxNode = helper.FindNearestKind(node, "throw_statement")
		if ctxNode == nil {
			return nil
		}
		return []ActionTarget{{RelType: model.Throw, TargetNode: node, ContextNode: ctxNode, Target: e.resolver.ResolveAction(gc, fCtx, node, model.Throw)}}

	case "explicit_constructor_stmt":
		ctxNode = node
		if ctxNode == nil {
			return nil
		}
		return []ActionTarget{
			{RelType: model.Call, TargetNode: node, ContextNode: ctxNode, Target: e.resolver.ResolveAction(gc, fCtx, node, model.Call)},
			{RelType: model.Create, TargetNode: node, ContextNode: ctxNode, Target: e.resolver.ResolveAction(gc, fCtx, node, model.Create)},
		}

	default:
		return nil
	}
}

// isUseRel 判定 Use 关系是否成立（排除定义点与调用/引用名称位置）。
func (e *Extractor) isUseRel(node coreast.Node, target *model.CodeElement) bool {
	// 1. 快速定位符号：如果不是变量或字段，直接 pass
	if target == nil || (target.Kind != model.Variable && target.Kind != model.Field) {
		return false
	}

	// 2. 排除定义点：通过父节点的 FieldName 判断
	// 如果当前 identifier 是其父节点的 "name" 字段，说明它是声明，不是使用
	parent := node.Parent()
	if nameNode := parent.ChildByFieldName("name"); nameNode != nil && nameNode.Equal(node) {
		return false
	}

	// 3. 排除特定语法噪声
	switch parent.Kind() {
	case "method_invocation", "method_reference":
		// 如果是方法调用/引用的名称部分，由 call_target 处理
		if nameNode := parent.ChildByFieldName("name"); nameNode != nil && nameNode.Equal(node) {
			return false
		}
	case "field_access":
		// 字段访问的字段名：外部字段/枚举常量由 field_atom 处理，此处排除避免重复；
		// 内部普通字段（CONSTANT 等）仍走 id_atom（field_atom 只放行 EnumConstant/外部符号）。
		if f := parent.ChildByFieldName("field"); f != nil && f.Equal(node) {
			if target.IsFormExternal || target.Kind == model.EnumConstant {
				return false
			}
		}
	case "scoped_identifier", "package_declaration", "import_declaration":
		return false
	case "lambda_expression":
		// 排除 lambda 参数声明：(request) -> request.getPath()
		// 参数声明位置不产 USE，仅接收者使用处产出。
		// 单参数 lambda 的 parameters 字段即 identifier 本身。
		if params := parent.ChildByFieldName("parameters"); params != nil && params.Equal(node) {
			return false
		}
	case "inferred_parameters":
		// 多参数无类型 lambda 的声明：(a, b) -> ...
		// inferred_parameters 仅出现在 lambda 中，其内 identifier 全是参数声明。
		return false
	}

	return true
}

// resolveTypeForElement 解析类型名，优先使用 Binder 已消解的作用域 QN（MethodReturnTypeWithQN），
// 避免短名（如嵌套 Builder）经 FindByShortName 命中错误的同名外层类。
func (e *Extractor) resolveTypeForElement(gc *core.GlobalContext, fCtx *core.FileContext, elem *model.CodeElement, raw string) *model.CodeElement {
	if elem != nil && elem.Extra != nil {
		if qn, ok := elem.Extra.Mores[constants.MethodReturnTypeWithQN].(string); ok && qn != "" {
			if entry, found := gc.FindByQualifiedName(qn); found {
				return entry.Element
			}
		}
	}
	return e.resolver.ResolveType(gc, fCtx, raw, model.Class)
}

// extractTypeFromParam 从参数原始文本中提取类型部分（PARAMETER 关系用）。
func (e *Extractor) extractTypeFromParam(p string) string {
	parts := strings.Fields(p)
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return p
}
