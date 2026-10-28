// Package symbol_resolver 实现 Java 语言的 core.SymbolResolver 接口。
// 职责：统一动作解析入口（三段式：Context -> Chain -> 合拍性校正 -> 兜底），
// 以及类型/变量/方法的基础解析（供 enricher/chain 内部共享）。
package symbol_resolver

import (
	"slices"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/resolver"
)

// SymbolResolver 实现 core.SymbolResolver。
type SymbolResolver struct{}

// New 创建 Java SymbolResolver。
func New() *SymbolResolver { return &SymbolResolver{} }

// =============================================================================
// core.SymbolResolver 接口实现
// =============================================================================

// BuildQualifiedName 拼接父符号 QN 与子符号名称为完整 QN。
func (r *SymbolResolver) BuildQualifiedName(parentQN, name string) string {
	if parentQN == "" || parentQN == "." {
		return name
	}
	return parentQN + "." + name
}

// RegisterPackage 注册包：将点号包名拆分为逐级包元素。
func (r *SymbolResolver) RegisterPackage(gc *core.GlobalContext, packageName string) {
	parts := strings.Split(packageName, ".")
	var current []string
	for _, part := range parts {
		if part == "" {
			continue
		}
		current = append(current, part)
		pkgQN := strings.Join(current, ".")
		if _, ok := gc.FindByQualifiedName(pkgQN); !ok {
			gc.AddDefinition(&core.DefinitionEntry{
				Element: &model.CodeElement{Kind: model.Package, Name: part, QualifiedName: pkgQN, IsFormSource: true},
			})
		}
	}
}

// IsPrimitive 判断类型名是否为 Java 基本类型。
func (r *SymbolResolver) IsPrimitive(typeName string) bool {
	switch typeName {
	case "int", "long", "short", "byte", "char", "boolean", "float", "double":
		return true
	}
	return false
}

// ResolveType 解析结构体符号，上下文未找到则返回指定 kind 的外部实体。
func (r *SymbolResolver) ResolveType(gc *core.GlobalContext, fc *core.FileContext, symbol string, kind model.ElementKind) *model.CodeElement {
	symbol = helper.Clean(symbol)
	if symbol == "" {
		return nil
	}

	if entries := helper.PreciseResolve(gc, fc, symbol); len(entries) > 0 {
		return entries[0].Element
	}

	// 找不到明确实体：按 import 表升级为外部 QN。
	qualifiedName := symbol
	if imps, ok := fc.Imports[symbol]; ok && len(imps) > 0 {
		qualifiedName = imps[0].RawImportPath
	}
	return &model.CodeElement{Name: symbol, QualifiedName: qualifiedName, Kind: kind, IsFormExternal: true}
}

// ResolveVar 处理变量查找（本地作用域回溯 + 类成员继承查找）。
func (r *SymbolResolver) ResolveVar(gc *core.GlobalContext, fc *core.FileContext, node coreast.Node, receiver, symbol string) *model.CodeElement {
	receiver = helper.Clean(receiver)
	symbol = helper.Clean(symbol)

	isStatic := false

	if receiver != "" {
		// 场景 A: this / super。
		if receiver == "this" || receiver == "super" {
			container := helper.GetBestElement(fc, node, []model.ElementKind{model.Class, model.AnonymousClass})
			if container == nil {
				return nil
			}
			isStatic = slices.Contains(container.Extra.Modifiers, "static")
			if receiver == "super" {
				return r.resolveFromInheritance(gc, fc, container, symbol, isStatic, container)
			}
			return r.resolveInScopeHierarchy(gc, fc, container.QualifiedName, symbol, isStatic, container)
		}

		// 场景 B: 类名静态访问。
		if entries := helper.PreciseResolve(gc, fc, receiver); len(entries) > 0 {
			receiverEle := entries[0].Element
			if receiverEle.Kind == model.Class || receiverEle.Kind == model.Interface || receiverEle.Kind == model.AnonymousClass {
				return r.resolveInScopeHierarchy(gc, fc, receiverEle.QualifiedName, symbol, true, receiverEle)
			}
		}

		// 场景 C: 跨对象访问 (data.age)。
		receiverVar := r.ResolveVar(gc, fc, node, "", receiver)
		if receiverVar != nil && receiverVar.Extra != nil {
			if typeQN, ok := receiverVar.Extra.Mores[constants.VariableTypeWithQN].(string); ok {
				if entries := helper.PreciseResolve(gc, fc, typeQN); len(entries) > 0 {
					receiverType := entries[0].Element
					if receiverType.Kind == model.Class || receiverType.Kind == model.Interface || receiverType.Kind == model.AnonymousClass {
						return r.resolveInScopeHierarchy(gc, fc, receiverType.QualifiedName, symbol, false, receiverVar)
					}
				}
			}
		}
	}

	// 无 receiver：按作用域链查找。
	container := helper.GetBestElement(fc, node, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})
	if container == nil {
		return nil
	}
	isStatic = slices.Contains(container.Extra.Modifiers, "static")
	return r.resolveInScopeHierarchy(gc, fc, container.QualifiedName, symbol, isStatic, container)
}

// ResolveFunc 处理方法查找：容器定位 -> 继承链搜索 -> 重载消解。
func (r *SymbolResolver) ResolveFunc(gc *core.GlobalContext, fc *core.FileContext, node coreast.Node, receiver, symbol string) *model.CodeElement {
	receiver = helper.Clean(receiver)
	symbol = helper.Clean(symbol)

	var container *model.CodeElement
	isStaticCall := false

	// 1. 确定搜索起始容器。
	if receiver != "" {
		if entries := helper.PreciseResolve(gc, fc, receiver); len(entries) > 0 {
			first := entries[0].Element
			switch first.Kind {
			case model.Class, model.Interface, model.Enum:
				container = first
				isStaticCall = true
			}
		}

		// this/super。
		if container == nil && (receiver == "this" || receiver == "super") {
			container = helper.GetBestElement(fc, node, []model.ElementKind{model.Class, model.AnonymousClass})
			if receiver == "super" && container != nil {
				if sc, ok := container.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
					if parents := helper.PreciseResolve(gc, fc, helper.Clean(sc)); len(parents) > 0 {
						container = parents[0].Element
					}
				}
			}
		}

		// 实例变量调用。
		if container == nil {
			if recvVar := r.ResolveVar(gc, fc, node, "", receiver); recvVar != nil {
				typeQN, _ := recvVar.Extra.Mores[constants.VariableTypeWithQN].(string)
				if typeQN == "" {
					typeQN, _ = recvVar.Extra.Mores[constants.VariableRawType].(string)
				}
				if typeQN != "" {
					if ents := helper.PreciseResolve(gc, fc, helper.Clean(typeQN)); len(ents) > 0 {
						container = ents[0].Element
					}
				}
			}
		}
	}

	// 无 receiver：从当前位置找最近类。
	if container == nil {
		container = helper.GetBestElement(fc, node, []model.ElementKind{model.Class, model.AnonymousClass})
	}
	if container == nil {
		return &model.CodeElement{Name: symbol, Kind: model.Method, IsFormExternal: true}
	}

	// 2. 实参信息（重载匹配）。
	argCount, inferredTypes := r.collectArgs(node, fc)

	// 3. 沿继承链搜索。
	mr := resolver.NewMemberResolver(gc, fc)
	if result := mr.ResolveMethod(container, symbol, argCount, inferredTypes, isStaticCall); result != nil {
		return result
	}

	// 4. 兜底外部符号。
	return &model.CodeElement{
		Name:           symbol,
		QualifiedName:  gc.BuildQualifiedName(container.QualifiedName, symbol),
		Kind:           model.Method,
		IsFormExternal: true,
	}
}

// ResolveAction 统一动作解析入口：三段式 + 合拍性校正 + 兜底。
func (r *SymbolResolver) ResolveAction(gc *core.GlobalContext, fc *core.FileContext, node coreast.Node, relType model.DependencyType) *model.CodeElement {
	if node == nil {
		return nil
	}

	// ① NodeContextResolver 提取双轨上下文。
	ctxResolver := resolver.NewNodeContextResolver()
	ctx := ctxResolver.Resolve(node, relType)
	if ctx.ExpressNode == nil {
		return nil
	}

	// ② ExpressionSegmenter 拉平符号链。
	seg := resolver.NewExpressionSegmenter(*fc.SourceBytes)
	chain := seg.Segment(ctx.ExpressNode, relType)
	if chain == nil {
		return nil
	}

	// ③ ChainResolver 沿链推导。
	argCount, inferredTypes := r.collectArgs(ctx.ExpressNode, fc)
	chainResolver := resolver.NewChainResolver(gc, fc)
	elem := chainResolver.ResolveChain(chain, relType, argCount, inferredTypes)
	if elem == nil {
		return r.generateContextualFallback(gc, fc, chain, relType)
	}

	// ④ 合拍性校正。
	aligned := r.alignElementWithRelType(gc, fc, elem, relType)
	if aligned != nil {
		return aligned
	}

	// ⑤ 兜底。
	return r.generateContextualFallback(gc, fc, chain, relType)
}

// =============================================================================
// 作用域链与继承查找
// =============================================================================

// resolveInScopeHierarchy 递归向上查找容器及继承链。
func (r *SymbolResolver) resolveInScopeHierarchy(gc *core.GlobalContext, fc *core.FileContext, previousQN, symbol string, isStatic bool, container *model.CodeElement) *model.CodeElement {
	if previousQN == "" {
		return nil
	}

	// 1. 当前层级直接匹配。
	targetQN := r.BuildQualifiedName(previousQN, symbol)
	if entry, ok := gc.FindByQualifiedName(targetQN); ok {
		if r.checkVisibility(gc, fc, container, entry) {
			isIllegalStatic := isStatic && entry.Element.Kind == model.Field && !slices.Contains(entry.Element.Extra.Modifiers, "static")
			if !isIllegalStatic {
				return entry.Element
			}
		}
	}

	previousEntry, ok := gc.FindByQualifiedName(previousQN)
	if !ok {
		return nil
	}

	// 2. 类/接口：递归继承链。
	if previousEntry.Element.Kind == model.Class || previousEntry.Element.Kind == model.Interface || previousEntry.Element.Kind == model.AnonymousClass {
		if inherited := r.resolveFromInheritance(gc, fc, previousEntry.Element, symbol, isStatic, container); inherited != nil {
			return inherited
		}
	}

	// 3. 递归上一级 Lexical Scope。
	return r.resolveInScopeHierarchy(gc, fc, previousEntry.ParentQN, symbol, isStatic, container)
}

// resolveFromInheritance 沿继承树查找字段/常量。
func (r *SymbolResolver) resolveFromInheritance(gc *core.GlobalContext, fc *core.FileContext, elem *model.CodeElement, symbol string, isStatic bool, source *model.CodeElement) *model.CodeElement {
	if elem == nil || elem.Extra == nil {
		return nil
	}

	var supers []string
	if sc, ok := elem.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
		supers = append(supers, sc)
	}
	if itfs, ok := elem.Extra.Mores[constants.ClassImplementedInterfaces].([]string); ok {
		supers = append(supers, itfs...)
	}

	for _, rawSuper := range supers {
		cleanSuper := strings.Split(rawSuper, "<")[0]
		if parents := helper.PreciseResolve(gc, fc, cleanSuper); len(parents) > 0 {
			parent := parents[0].Element
			targetQN := r.BuildQualifiedName(parent.QualifiedName, symbol)
			if entry, ok := gc.FindByQualifiedName(targetQN); ok {
				if r.checkVisibility(gc, fc, source, entry) {
					if !isStatic || slices.Contains(entry.Element.Extra.Modifiers, "static") {
						return entry.Element
					}
				}
			}
			if found := r.resolveFromInheritance(gc, fc, parent, symbol, isStatic, source); found != nil {
				return found
			}
		}
	}
	return nil
}

// checkVisibility 可见性检查。
func (r *SymbolResolver) checkVisibility(gc *core.GlobalContext, fc *core.FileContext, container *model.CodeElement, target *core.DefinitionEntry) bool {
	if target.Element.Kind == model.Variable {
		return true
	}

	containerOuter := helper.GetOutermostClassQN(container.QualifiedName)
	targetOuter := helper.GetOutermostClassQN(target.Element.QualifiedName)
	if containerOuter != "" && containerOuter == targetOuter {
		return true
	}

	if target.Element.Extra == nil || target.Element.Extra.Modifiers == nil {
		return false
	}
	mods := target.Element.Extra.Modifiers
	if slices.Contains(mods, "public") {
		return true
	}

	targetPkg := helper.GetRealPackage(gc, target.Element)
	if targetPkg == fc.PackageName {
		return true
	}

	if slices.Contains(mods, "protected") {
		sourceClass := helper.GetOwnerClassQN(gc, container)
		return helper.IsSubClassOf(gc, fc, sourceClass, target.ParentQN)
	}

	return false
}

// =============================================================================
// 实参收集与重载推断
// =============================================================================

// collectArgs 从调用节点收集实参数量与推断类型。
func (r *SymbolResolver) collectArgs(node coreast.Node, fc *core.FileContext) (int, []string) {
	if node == nil {
		return 0, nil
	}
	invNode := r.findInvocationNode(node)
	if invNode == nil {
		return 0, nil
	}
	args := invNode.ChildByFieldName("arguments")
	if args == nil {
		return 0, nil
	}
	argCount := args.NamedChildCount()
	types := make([]string, 0, argCount)
	src := *fc.SourceBytes
	for i := 0; i < argCount; i++ {
		types = append(types, r.inferArgType(args.NamedChild(i), src))
	}
	return argCount, types
}

// inferArgType 推断单个实参的大致类型。
func (r *SymbolResolver) inferArgType(arg coreast.Node, src []byte) string {
	if arg == nil {
		return "unknown"
	}
	switch arg.Kind() {
	case "string_literal":
		return "String"
	case "decimal_integer_literal", "hex_integer_literal":
		return "int"
	case "decimal_floating_point_literal":
		return "double"
	case "true", "false":
		return "boolean"
	case "null_literal":
		return "null"
	case "object_creation_expression", "cast_expression":
		if t := arg.ChildByFieldName("type"); t != nil {
			return t.Utf8Text(src)
		}
		return "unknown"
	case "array_creation_expression":
		if t := arg.ChildByFieldName("type"); t != nil {
			return t.Utf8Text(src) + "[]"
		}
		return "unknown"
	default:
		return "unknown"
	}
}

// findInvocationNode 向上查找调用节点。
func (r *SymbolResolver) findInvocationNode(n coreast.Node) coreast.Node {
	for curr := n; curr != nil; curr = curr.Parent() {
		k := curr.Kind()
		if k == "method_invocation" || k == "object_creation_expression" || k == "explicit_constructor_invocation" {
			return curr
		}
		if strings.HasSuffix(k, "_statement") {
			break
		}
	}
	return nil
}

// =============================================================================
// 合拍性校正与兜底
// =============================================================================

// alignElementWithRelType 校验推导结果是否符合关系类型预期，必要时语义修正。
func (r *SymbolResolver) alignElementWithRelType(gc *core.GlobalContext, fc *core.FileContext, elem *model.CodeElement, relType model.DependencyType) *model.CodeElement {
	if elem == nil {
		return nil
	}

	switch relType {
	case model.Call:
		// Call 期望 Method。若得 Class/Interface -> 未注册构造的实例化，构建外部 Method。
		switch elem.Kind {
		case model.Method, model.MethodRef, model.Lambda:
			return elem
		case model.Class, model.Interface, model.AnonymousClass, model.Enum:
			return &model.CodeElement{
				Name:           elem.Name,
				QualifiedName:  elem.QualifiedName,
				Kind:           model.Method,
				IsFormExternal: elem.IsFormExternal,
			}
		}

	case model.Use, model.Assign:
		// 期望 Field / Variable / EnumConstant。
		switch elem.Kind {
		case model.Field, model.Variable, model.EnumConstant:
			return elem
		}

	case model.Cast, model.Create, model.Throw, model.Return, model.Parameter:
		// 期望类型。若得 Field/Variable -> 提取声明类型；Method -> 提取返回值。
		switch elem.Kind {
		case model.Class, model.Interface, model.AnonymousClass, model.Enum, model.KAnnotation:
			return elem
		case model.Field, model.Variable:
			if elem.Extra != nil {
				if raw, ok := elem.Extra.Mores[constants.FieldRawType].(string); ok {
					return r.ResolveType(gc, fc, raw, model.Class)
				}
				if raw, ok := elem.Extra.Mores[constants.VariableRawType].(string); ok {
					return r.ResolveType(gc, fc, raw, model.Class)
				}
			}
		case model.Method:
			if elem.Extra != nil {
				if raw, ok := elem.Extra.Mores[constants.MethodReturnType].(string); ok {
					return r.ResolveType(gc, fc, raw, model.Class)
				}
			}
		}
	}

	return elem
}

// generateContextualFallback 依据拉平链拼装最吻合的外部符号 QN。
func (r *SymbolResolver) generateContextualFallback(gc *core.GlobalContext, fc *core.FileContext, chain *resolver.ExpressionChain, relType model.DependencyType) *model.CodeElement {
	if chain == nil {
		return nil
	}

	// 头部基底。
	base := chain.Head.Name
	if chain.Head.Type == resolver.HeadCastExpr && chain.Head.CastType != "" {
		base = chain.Head.CastType
	}
	if chain.Head.Type == resolver.HeadNewExpr && chain.Head.Name != "" {
		base = chain.Head.Name
	}
	if base == "" {
		return nil
	}
	base = helper.Clean(base)

	// 查 import 表升级为完整 QN。
	qualified := base
	if imps, ok := fc.Imports[base]; ok && len(imps) > 0 {
		qualified = imps[0].RawImportPath
	}

	// 累加 Segments。按关系类型决定期望的元素种类。
	kind := model.Class
	switch relType {
	case model.Call:
		kind = model.Method
	case model.Use, model.Assign:
		kind = model.Field
	}

	for _, seg := range chain.Segments {
		if seg.Kind == resolver.SegmentClass {
			qualified += "$" + seg.Name
		} else if seg.Kind == resolver.SegmentMethod && kind == model.Method {
			qualified = gc.BuildQualifiedName(qualified, seg.Name)
		} else if seg.Kind == resolver.SegmentField && kind == model.Field {
			qualified = gc.BuildQualifiedName(qualified, seg.Name)
		}
	}

	return &model.CodeElement{
		Name:           base,
		QualifiedName:  qualified,
		Kind:           kind,
		IsFormExternal: true,
	}
}
