package resolver

import (
	"slices"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	coreast "github.com/CodMac/arch-lens-dep-analyer/core/ast"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

// ChainResolver 沿拉平链逐段在符号库中推导目标符号。
// 维护 currType 供下一段解析；断链时优雅回退 lastGood。
type ChainResolver struct {
	gc *core.GlobalContext
	fc *core.FileContext

	// 泛型元素类型传播：方法返回类型中提取 List<Order> -> Order。
	genericElem map[string]string

	// instanceCtx 标记当前接收者为实例上下文（new 表达式 / this / super / 变量 / cast 后调用）。
	// 为 true 时，后续段在 Class/Interface/Enum 接收者上的方法/字段按实例调用解析
	//（isStatic=false），避免把 new Builder().name() 误判为静态访问。
	instanceCtx bool
}

// NewChainResolver 创建链式解析器。
func NewChainResolver(gc *core.GlobalContext, fc *core.FileContext) *ChainResolver {
	return &ChainResolver{gc: gc, fc: fc, genericElem: make(map[string]string)}
}

// ResolveChain 从 Head 开始，沿链解析。
// relType 用于最终合拍性校正（Call/Use/Assign 等）。
func (cr *ChainResolver) ResolveChain(chain *ExpressionChain, relType model.DependencyType, argCount int, inferredTypes []string) *model.CodeElement {
	if chain == nil {
		return nil
	}

	// 1. 处理 Head，并据此初始化实例上下文。
	cr.instanceCtx = false
	curr := cr.resolveHead(chain.Head, relType, argCount, inferredTypes, len(chain.Segments) > 0)
	if curr == nil {
		return nil
	}
	cr.seedInstanceCtx(chain.Head, curr)

	// 2. 迭代 Segments。
	lastGood := curr
	for _, seg := range chain.Segments {
		next := cr.resolveSegment(curr, seg, argCount, inferredTypes)
		if next == nil {
			// 断链：回退 lastGood。
			return lastGood
		}
		lastGood = next
		curr = next
		// 方法/字段段解析成功后，后续段仍是实例上下文（链式接收者）。
		cr.instanceCtx = seg.Kind != SegmentClass
	}

	return lastGood
}

// seedInstanceCtx 依据头部类型与解析结果初始化 instanceCtx。
func (cr *ChainResolver) seedInstanceCtx(head ExpressionHead, resolved *model.CodeElement) {
	switch head.Type {
	case HeadNewExpr, HeadThis, HeadSuper, HeadImplicitMethod, HeadCastExpr, HeadCtorInvoke:
		// new X(...)、this、super、隐式 this.method()、cast 后调用、构造器调用 -> 实例上下文。
		cr.instanceCtx = true
	case HeadIdent:
		// 标识符解析为变量/字段 -> 实例接收者；解析为类型 -> 静态访问。
		cr.instanceCtx = resolved != nil && (resolved.Kind == model.Variable || resolved.Kind == model.Field)
	case HeadType:
		// 显式类型名 -> 静态访问。
		cr.instanceCtx = false
	}
}

// resolveHead 解析头部符号，返回当前类型对应的 CodeElement。
// argCount/inferredTypes 供隐式方法调用头的重载消解使用；hasSegments 标记链上是否存在后续段
// （用于区分独立构造调用与匿名类实例化的链式调用）。
func (cr *ChainResolver) resolveHead(head ExpressionHead, relType model.DependencyType, argCount int, inferredTypes []string, hasSegments bool) *model.CodeElement {
	switch head.Type {
	case HeadThis:
		return cr.currentClass(head.ASTNode)

	case HeadSuper:
		cls := cr.currentClass(head.ASTNode)
		if cls == nil {
			return nil
		}
		// super -> 父类。
		if sc, ok := cls.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
			if parents := helper.PreciseResolve(cr.gc, cr.fc, helper.Clean(sc)); len(parents) > 0 {
				return parents[0].Element
			}
		}
		return nil

	case HeadNewExpr:
		// 匿名类实例化的链式调用：new Runnable(){...}.run() 应解析到匿名类自身，
		// 而非接口类型（否则后续 .run() 段会在 Runnable 接口上查找，丢失匿名类方法）。
		// 仅当链上存在后续段（链式调用）且为 Call 关系时启用；
		// 独立 new X(){...}（无后续段）是构造调用，仍解析到构造类型本身（如 Runnable）。
		if relType == model.Call && hasSegments && head.ASTNode != nil && hasClassBodyNode(head.ASTNode) {
			if anon := cr.currentAnonClass(head.ASTNode); anon != nil {
				return anon
			}
		}
		return cr.resolveTypeFromName(head.Name)

	case HeadCastExpr:
		return cr.resolveTypeFromName(head.CastType)

	case HeadImplicitMethod:
		// 无接收者的隐式方法调用（this.method()）。
		cls := cr.currentClass(head.ASTNode)
		if cls == nil {
			return nil
		}
		mr := NewMemberResolver(cr.gc, cr.fc)
		if resolved := mr.ResolveMethod(cls, head.Name, argCount, inferredTypes, false); resolved != nil {
			return resolved
		}
		// 匿名类/局部类内未命中：沿词法外层类回溯解析。
		// 例：匿名类 run() 内调用外部类方法 simpleMethod()，currentClass 返回匿名类，
		// 本类/父类均无该方法，需回溯到外部类 CallRelationSuite 查找。
		for outer := cr.outerClassOf(cls); outer != nil; outer = cr.outerClassOf(outer) {
			if resolved := mr.ResolveMethod(outer, head.Name, argCount, inferredTypes, false); resolved != nil {
				return resolved
			}
		}
		return nil

	case HeadIdent:
		// 可能是变量、字段、类名。
		return cr.resolveIdent(head)

	case HeadType:
		return cr.resolveTypeFromName(head.Name)

	case HeadCtorInvoke:
		return cr.resolveCtorInvoke(head, argCount, inferredTypes)
	}
	return nil
}

// resolveSegment 解析链中的一段。
func (cr *ChainResolver) resolveSegment(curr *model.CodeElement, seg ExpressionSegment, argCount int, inferredTypes []string) *model.CodeElement {
	if curr == nil {
		return nil
	}

	mr := NewMemberResolver(cr.gc, cr.fc)

	// 链式方法接收者：方法段的后续段应在方法返回类型上解析成员。
	// 例：new Builder().name().age() 中，name() 段解析后返回 Method(name)，
	// 下一段 age 应在其返回类型 Builder 上查找，而非 Method 元素自身。
	if curr.Kind == model.Method {
		if container := cr.methodReturnContainer(curr); container != nil {
			switch seg.Kind {
			case SegmentMethod:
				if resolved := mr.ResolveMethod(container, seg.Name, argCount, inferredTypes, false); resolved != nil {
					cr.propagateGenericElement(resolved)
					return resolved
				}
			case SegmentField:
				if resolved := mr.ResolveField(container, seg.Name, false); resolved != nil {
					return resolved
				}
			}
		}
	}

	// 实例变量/字段接收者：先按声明类型解析成员，再回退到变量自身 QN。
	// 例：user.getName() 中 curr=Variable(user)，声明类型 User -> 在 User 上查 getName。
	if curr.Kind == model.Variable || curr.Kind == model.Field {
		if container := cr.receiverContainer(curr); container != nil {
			switch seg.Kind {
			case SegmentMethod:
				if resolved := mr.ResolveMethod(container, seg.Name, argCount, inferredTypes, false); resolved != nil {
					cr.propagateGenericElement(resolved)
					return resolved
				}
			case SegmentField:
				if resolved := mr.ResolveField(container, seg.Name, false); resolved != nil {
					return resolved
				}
			}
		}
	}

	switch seg.Kind {
	case SegmentField:
		// 静态字段访问（curr 是类型且非实例上下文）或实例字段访问。
		isStatic := !cr.instanceCtx && (curr.Kind == model.Class || curr.Kind == model.Interface || curr.Kind == model.Enum)
		return mr.ResolveField(curr, seg.Name, isStatic)

	case SegmentMethod:
		// 方法调用：curr 为类型且处于静态上下文（类型名/静态链）时按静态解析，
		// 否则（new 表达式/变量/cast 后调用等实例接收者）按实例解析。
		isStatic := !cr.instanceCtx && (curr.Kind == model.Class || curr.Kind == model.Interface || curr.Kind == model.Enum)
		resolved := mr.ResolveMethod(curr, seg.Name, argCount, inferredTypes, isStatic)
		if resolved == nil {
			// 兜底：构造外部方法符号。
			return model.NewExternalElement(model.Method, seg.Name, cr.gc.BuildQualifiedName(curr.QualifiedName, seg.Name))
		}

		// 泛型元素类型传播：解析返回值类型并缓存。
		cr.propagateGenericElement(resolved)
		return resolved

	case SegmentArray:
		// 数组下标读取：消费泛型元素类型。
		if elemType, ok := cr.genericElem[curr.QualifiedName]; ok {
			return cr.resolveTypeFromName(elemType)
		}
		return curr

	case SegmentClass:
		// 嵌套类访问。
		return cr.resolveTypeFromName(cr.gc.BuildQualifiedName(curr.QualifiedName, seg.Name))
	}
	return nil
}

// resolveCtorInvoke 解析显式构造器调用（super() / this() / TypeName(...)）。
// 定位目标类型后，沿继承链查找同名构造器（含重载消解）。
func (cr *ChainResolver) resolveCtorInvoke(head ExpressionHead, argCount int, inferredTypes []string) *model.CodeElement {
	ref := head.CtorRef
	if ref == "" {
		return nil
	}

	// 1. 定位目标类型元素。
	var targetType *model.CodeElement
	switch ref {
	case "super":
		// super() -> 当前类 -> 父类。
		cls := cr.currentClass(head.ASTNode)
		if cls == nil {
			return nil
		}
		if sc, ok := cls.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
			if parents := helper.PreciseResolve(cr.gc, cr.fc, helper.Clean(sc)); len(parents) > 0 {
				targetType = parents[0].Element
			}
		}
		if targetType == nil {
			return nil
		}
	case "this":
		// this() -> 当前类。
		targetType = cr.currentClass(head.ASTNode)
		if targetType == nil {
			return nil
		}
	default:
		// TypeName(...) -> 按类型名解析（可能是同文件/同包/import 类型）。
		targetType = cr.resolveTypeFromName(ref)
		if targetType == nil || !isTypeKind(targetType.Kind) {
			return nil
		}
	}

	// 2. 沿类型及其父类链查找构造器。
	mr := NewMemberResolver(cr.gc, cr.fc)
	ctorName := targetType.Name
	if elem := cr.resolveCtorOnType(mr, targetType, ctorName, argCount, inferredTypes); elem != nil {
		return elem
	}

	// 3. 兜底：构造外部构造器符号（QN=类型QN.构造器名）。
	return model.NewExternalElement(model.Method, ctorName, cr.gc.BuildQualifiedName(targetType.QualifiedName, ctorName))
}

// resolveCtorOnType 在目标类型及其父类链上查找构造器。
func (cr *ChainResolver) resolveCtorOnType(mr *MemberResolver, elem *model.CodeElement, ctorName string, argCount int, inferredTypes []string) *model.CodeElement {
	if elem == nil || elem.Extra == nil {
		return nil
	}

	// 当前类型查找构造器。
	if resolved := mr.ResolveMethod(elem, ctorName, argCount, inferredTypes, false); resolved != nil {
		return resolved
	}

	// 沿父类递归（构造器不继承，但 super() 需在父类链上找，此路径兜底覆盖显式父类构造器）。
	if sc, ok := elem.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
		if parents := helper.PreciseResolve(cr.gc, cr.fc, helper.Clean(sc)); len(parents) > 0 {
			if found := cr.resolveCtorOnType(mr, parents[0].Element, ctorName, argCount, inferredTypes); found != nil {
				return found
			}
		}
	}
	return nil
}

// resolveIdent 解析标识符头部：优先变量/字段，其次类名。
func (cr *ChainResolver) resolveIdent(head ExpressionHead) *model.CodeElement {
	symbol := head.Name
	if symbol == "" {
		return nil
	}

	// 1. 尝试作为变量/字段（当前作用域链）。
	if ele := cr.resolveVarInScope(head.ASTNode, symbol); ele != nil {
		return ele
	}

	// 2. 尝试作为类型名兜底。
	// 注意：PreciseResolve 的 FindByShortName 会把字段/变量短名当类型命中返回
	//（Builder 链式调用等依赖此回退）。因此不能仅接受类型 kind 一刀切拦截，
	// 否则会破坏链式字段/方法解析。但静态上下文中引用实例字段是非法访问
	//（ResolveVar 已拦截返回 NIL），此处兜底必须同样拒绝，避免绕过检查。
	if typeEle := cr.resolveTypeFromName(symbol); typeEle != nil {
		if isTypeKind(typeEle.Kind) {
			return typeEle
		}
		// 兜底命中的是非类型（Field/Variable）。
		// 静态上下文拦截仅适用于：裸标识符引用（无显式 receiver 的字段访问）且
		// 目标是当前类自身的实例字段（如静态方法内引用 instanceVar）。
		// 链式调用（new Builder().name()）的 name 段属于 Builder 类，不受此拦截。
		if typeEle.Kind == model.Field && !cr.hasExplicitReceiver(head.ASTNode) &&
			cr.isInStaticContext(head.ASTNode) && cr.isFieldOfCurrentClass(head.ASTNode, typeEle) &&
			typeEle.Extra != nil && !slices.Contains(typeEle.Extra.Modifiers, "static") {
			// 返回 Class kind 的外部元素（而非 nil）：
			// 1) 使 ResolveAction 不落入 generateContextualFallback（其对 Use 兜底为 Field，
			//    会通过 isUseRel 过滤产出非法 USE 关系）；
			// 2) Class kind 会被 isUseRel 过滤（非 Variable/Field），USE 关系不产出，
			//    实现“静态方法内引用实例字段应解析失败”的语义。
			return model.NewExternalElement(model.Class, symbol, cr.contextualQN(symbol))
		}
		return typeEle
	}

	// 3. 外部兜底。
	return model.NewExternalElement(model.Class, symbol, cr.contextualQN(symbol))
}

// resolveVarInScope 在作用域链中查找变量/字段。
func (cr *ChainResolver) resolveVarInScope(node coreast.Node, symbol string) *model.CodeElement {
	// 委托给 SymbolResolver 的 ResolveVar（通过 gc.Resolver）。
	return cr.gc.Resolver.ResolveVar(cr.gc, cr.fc, node, "", symbol)
}

// isTypeKind 判断 ElementKind 是否为类型类（可作为类型解析结果）。
func isTypeKind(k model.ElementKind) bool {
	switch k {
	case model.Class, model.Interface, model.Enum, model.KAnnotation, model.AnonymousClass:
		return true
	}
	return false
}

// receiverContainer 依据变量/字段的声明类型解析接收者容器（类/接口/枚举）。
// 优先用带 QN 的类型（java.variable.type_with_qn / java.field.raw_type 需二次解析），
// 找不到则回退 nil（由调用方继续按变量自身 QN 兜底）。
func (cr *ChainResolver) receiverContainer(elem *model.CodeElement) *model.CodeElement {
	if elem == nil || elem.Extra == nil {
		return nil
	}

	typeQN, _ := elem.Extra.Mores[constants.VariableTypeWithQN].(string)
	if typeQN == "" {
		typeQN, _ = elem.Extra.Mores[constants.FieldRawType].(string)
	}
	if typeQN == "" {
		return nil
	}

	container := cr.resolveTypeFromName(typeQN)
	if container == nil || !isTypeKind(container.Kind) {
		return nil
	}
	return container
}

// methodReturnContainer 依据方法的返回类型解析接收者容器（类/接口/枚举）。
// 用于链式调用：方法段的后续段在返回类型上解析成员（如 new Builder().name().age()）。
func (cr *ChainResolver) methodReturnContainer(elem *model.CodeElement) *model.CodeElement {
	if elem == nil || elem.Extra == nil {
		return nil
	}

	retQN, _ := elem.Extra.Mores[constants.MethodReturnTypeWithQN].(string)
	if retQN == "" {
		retQN, _ = elem.Extra.Mores[constants.MethodReturnType].(string)
	}
	if retQN == "" || retQN == "void" {
		return nil
	}

	container := cr.resolveTypeFromName(retQN)
	if container == nil || !isTypeKind(container.Kind) {
		return nil
	}
	return container
}

// resolveTypeFromName 将类型名解析为类型元素。
func (cr *ChainResolver) resolveTypeFromName(name string) *model.CodeElement {
	name = helper.Clean(name)
	if name == "" {
		return nil
	}
	return cr.gc.Resolver.ResolveType(cr.gc, cr.fc, name, model.Class)
}

// hasExplicitReceiver 判断该标识符是否作为字段/方法访问的接收者目标（x.name / x.name()）。
// 若 identifier 是 field_access / method_invocation 的 name 字段，则属于链式访问，
// 不做静态上下文拦截（receiver 是新对象/变量，非静态 this）。
func (cr *ChainResolver) hasExplicitReceiver(node coreast.Node) bool {
	if node == nil {
		return false
	}
	p := node.Parent()
	if p == nil {
		return false
	}
	switch p.Kind() {
	case "field_access", "method_invocation", "method_reference":
		nameNode := p.ChildByFieldName("name")
		if nameNode != nil && nameNode.Equal(node) {
			return true
		}
	}
	return false
}

// isFieldOfCurrentClass 判断字段是否属于当前代码所在类。
func (cr *ChainResolver) isFieldOfCurrentClass(node coreast.Node, field *model.CodeElement) bool {
	cls := cr.currentClass(node)
	if cls == nil || field == nil {
		return false
	}
	owner := helper.GetOwnerClassQN(cr.gc, field)
	return owner != "" && owner == cls.QualifiedName
}

// isInStaticContext 判断当前代码是否位于静态上下文（静态方法/静态块/静态字段初始化）。
func (cr *ChainResolver) isInStaticContext(node coreast.Node) bool {
	container := helper.GetBestElement(cr.fc, node, []model.ElementKind{model.Method, model.ScopeBlock, model.Field})
	if container != nil && container.Extra != nil {
		return slices.Contains(container.Extra.Modifiers, "static")
	}
	// 兜底：无静态容器则视为非静态上下文（保守返回 false）。
	return false
}

// currentClass 定位当前代码所在的类。
func (cr *ChainResolver) currentClass(node coreast.Node) *model.CodeElement {
	if node == nil {
		return nil
	}
	return helper.GetBestElement(cr.fc, node, []model.ElementKind{model.Class, model.Interface, model.AnonymousClass, model.Enum})
}

// currentAnonClass 定位 new 表达式对应的匿名类元素。
func (cr *ChainResolver) currentAnonClass(node coreast.Node) *model.CodeElement {
	if node == nil {
		return nil
	}
	return helper.GetBestElement(cr.fc, node, []model.ElementKind{model.AnonymousClass})
}

// hasClassBodyNode 判断 object_creation_expression 是否携带 class_body（匿名类）。
func hasClassBodyNode(node coreast.Node) bool {
	if node == nil || node.Kind() != "object_creation_expression" {
		return false
	}
	for i := 0; i < node.NamedChildCount(); i++ {
		if node.NamedChild(i).Kind() == "class_body" {
			return true
		}
	}
	return false
}

// outerClassOf 依据元素在符号库中的父级关系回溯词法外层类。
// 匿名类/局部类的 ParentQN 指向其声明所在的外层方法/类，逐级上溯直到类元素。
func (cr *ChainResolver) outerClassOf(elem *model.CodeElement) *model.CodeElement {
	if elem == nil || elem.QualifiedName == "" {
		return nil
	}
	entry, ok := cr.gc.FindByQualifiedName(elem.QualifiedName)
	if !ok {
		return nil
	}
	prevQN := entry.ParentQN
	for prevQN != "" {
		if parent, ok := cr.gc.FindByQualifiedName(prevQN); ok {
			if isTypeKind(parent.Element.Kind) {
				return parent.Element
			}
			prevQN = parent.ParentQN
			continue
		}
		break
	}
	return nil
}

// propagateGenericElement 从方法返回类型提取泛型元素类型并缓存到 genericElem。
// 形如 List<Order> getOrders() -> genericElem[methodQN] = Order。
func (cr *ChainResolver) propagateGenericElement(method *model.CodeElement) {
	if method.Extra == nil {
		return
	}
	rt, _ := method.Extra.Mores[constants.MethodReturnType].(string)
	if !strings.Contains(rt, "<") {
		return
	}
	start, end := strings.Index(rt, "<"), strings.LastIndex(rt, ">")
	if start == -1 || end == -1 || start >= end {
		return
	}
	inner := rt[start+1 : end]
	if strings.Contains(inner, "<") {
		// 嵌套泛型取最内层？这里取首个顶层实参即可。
		inner = strings.Split(inner, ",")[0]
	}
	inner = helper.Clean(inner)
	if inner == "" {
		return
	}
	// 仅当该元素类型在符号库中命中才传播（避免传播未知类型）。
	// 用 PreciseResolve 支持短名（同文件/同包/嵌套类）解析，而非仅限精确 QN。
	if len(helper.PreciseResolve(cr.gc, cr.fc, inner)) > 0 || helper.IsBoxedPrimitive(inner) {
		cr.genericElem[method.QualifiedName] = inner
	}
}

// contextualQN 依据 import 表拼装外部符号 QN。
func (cr *ChainResolver) contextualQN(symbol string) string {
	if imps, ok := cr.fc.Imports[symbol]; ok && len(imps) > 0 {
		return imps[0].RawImportPath
	}
	return symbol
}
