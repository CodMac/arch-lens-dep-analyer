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
	gc core.GlobalContext
	fc core.FileContext

	// 泛型元素类型传播：方法返回类型中提取 List<Order> -> Order。
	genericElem map[string]string

	// instanceCtx 标记当前接收者为实例上下文（new 表达式 / this / super / 变量 / cast 后调用）。
	// 为 true 时，后续段在 Class/Interface/Enum 接收者上的方法/字段按实例调用解析
	//（isStatic=false），避免把 new Builder().name() 误判为静态访问。
	instanceCtx bool
}

// NewChainResolver 创建链式解析器。
func NewChainResolver(gc core.GlobalContext, fc core.FileContext) *ChainResolver {
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
	// HeadClassLit 带后续段：Xxx.class.getName() 的段在 java.lang.Class 上解析成员，
	// 而非类型自身（修复 .class. receiver 精度问题，ClassUtils L218 等）。
	// 纯 Xxx.class（无后续段）保持类型元素（作为 USE/Create 目标是类型引用）。
	if chain.Head.Type == HeadClassLit && len(chain.Segments) > 0 {
		curr = cr.classLiteralContainer()
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
	case HeadClassLit:
		// 类字面量 Xxx.class：无段时为类型引用（静态）；带段时接收者为 java.lang.Class
		// 实例（getCanonicalName() 等实例方法），由 ResolveChain 在覆盖 curr 后置位。
		cr.instanceCtx = true
	}
}

// resolveHead 解析头部符号，返回当前类型对应的 CodeElement。
// argCount/inferredTypes 供隐式方法调用头的重载消解使用；hasSegments 标记链上是否存在后续段
// （用于区分独立构造调用与匿名类实例化的链式调用）。
func (cr *ChainResolver) resolveHead(head ExpressionHead, relType model.DependencyType, argCount int, inferredTypes []string, hasSegments bool) *model.CodeElement {
	switch head.Type {
	case HeadThis:
		cls := cr.currentClass(head.ASTNode)
		if cls == nil {
			return nil
		}
		// C-7 限定 this：X.this 的 Name 携带限定类型名 X（非裸 "this"）。
		// 优先解析到限定类型 X，使后续方法/字段段在 X 的实例成员上解析；
		// X 未解析（外部/未注册）时回退当前类，保证链不断裂。
		if head.Name != "" && head.Name != "this" {
			if typeEle := cr.resolveTypeFromNameAt(head.ASTNode, head.Name); typeEle != nil && isTypeKind(typeEle.Kind) {
				return typeEle
			}
			return cls
		}
		return cls

	case HeadSuper:
		cls := cr.currentClass(head.ASTNode)
		if cls == nil {
			return nil
		}
		// super -> 父类。
		if sc, ok := cls.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
			if parents := helper.ResolveInheritanceParent(cr.gc, cr.fc, cls, helper.Clean(sc)); len(parents) > 0 {
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
		typeElem := cr.resolveTypeStrictAt(head.ASTNode, head.Name)
		// 独立构造调用（无后续段）：解析到真实构造器元素，QN = 类QN.类名(参数类型列表)。
		// 否则会退化为类 QN（再被 enricher 补成 "类QN()"），与 collector 产出的构造器
		// 元素 QN 对不上，导致关系目标悬空。
		if relType == model.Call && !hasSegments && typeElem != nil &&
			!typeElem.IsFormExternal && isTypeKind(typeElem.Kind) {
			return cr.resolveConstructor(typeElem, argCount, inferredTypes)
		}
		return typeElem

	case HeadCastExpr:
		return cr.resolveTypeStrictAt(head.ASTNode, head.CastType)

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
		return cr.resolveTypeStrictAt(head.ASTNode, head.Name)

	case HeadClassLit:
		// Xxx.class 头部：解析为类型元素本身。带后续段时由 ResolveChain 覆盖为
		// java.lang.Class 容器（成员解析），纯 class 引用保持类型元素。
		return cr.resolveTypeStrictAt(head.ASTNode, head.Name)

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
		if container := cr.methodReturnContainer(curr, seg.ASTNode); container != nil {
			switch seg.Kind {
			case SegmentMethod:
				if resolved := mr.ResolveMethod(container, seg.Name, argCount, inferredTypes, false); resolved != nil {
					cr.propagateGenericElement(resolved)
					return resolved
				}
				// 外部返回类型接收者兜底（C-3 #95）：仅当 curr 为外部方法（如 varGenericElem 传播
				// 返回类型的 Supplier.get）时，用接收者容器 QN 构造外部方法符号，避免落入底部
				// switch 用方法自身 QN 拼出被污染的裸名（Supplier.get.forExecutables()）。
				// 内部方法返回外部类型（如 getList().add）保持旧断链语义（方法 QN 链），不在此兜底。
				if curr.IsFormExternal {
					return model.NewExternalElement(model.Method, seg.Name, cr.gc.BuildQualifiedName(container.QualifiedName, seg.Name))
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
		// 方法引用且声明类型不可知（lambda 形参 inferred / var）：无法确定成员，
		// 交由底部 SegmentMethod 分支走「词法外层 + 短名」兜底，
		// 避免在此按外部类型构造 `inferred.getGroup()` 伪符号。
		if !(seg.IsMethodRef && cr.receiverTypeUnknown(curr)) {
			if container := cr.receiverContainer(curr, seg.ASTNode); container != nil {
				switch seg.Kind {
				case SegmentMethod:
					if resolved := mr.ResolveMethod(container, seg.Name, argCount, inferredTypes, false); resolved != nil {
						cr.propagateGenericElement(resolved)
						return resolved
					}
					// C-3 #95：接收者字段/变量类型为单类型参数容器 C<X>（如 Supplier<X>）上的
					// get 型方法（如 .get()）语义上返回类型参数 X。将 X 作为外部方法返回类型写入，
					// 使后续段（.getArgumentsForConstraint()）的 methodReturnContainer 能在 X 上继续
					// 解析，而非断链为 Supplier.get.getArgumentsForConstraint() 外部裸名。
					// 仅依赖源码内类型文本的泛型实参，不建外部类型签名映射表。
					// 收窄条件（spring 全量 100 文件验证）：仅「get 开头方法 + 单类型参数容器」
					// 才传播——List<X>.stream()/Map<K,V>.values()/Set<X>.iterator() 等包装型
					// 方法返回非 X，若误传播会把后续段错误解析到 X 上（泛型实参污染）。
					if elemType := cr.varGenericElem(curr); elemType != "" &&
						strings.HasPrefix(seg.Name, "get") && cr.singleTypeArg(curr) {
						ext := model.NewExternalElement(model.Method, seg.Name, cr.gc.BuildQualifiedName(container.QualifiedName, seg.Name))
						ext.GetMores()[constants.MethodReturnType] = elemType
						return ext
					}
					// 外部类型接收者：ResolveMethod 无成员定义，改用接收者类型 QN 构造外部方法符号。
					// 例：exchange.getRequest() -> org.springframework.web.server.ServerWebExchange.getRequest()，
					// 而非 ...lambda$1.exchange.getRequest()（变量 QN 污染）。
					return model.NewExternalElement(model.Method, seg.Name, cr.gc.BuildQualifiedName(container.QualifiedName, seg.Name))
				case SegmentField:
					if resolved := mr.ResolveField(container, seg.Name, false); resolved != nil {
						return resolved
					}
				}
			}
		}
	}

	switch seg.Kind {
	case SegmentField:
		// 静态字段访问（curr 是类型且非实例上下文）或实例字段访问。
		isStatic := !cr.instanceCtx && (curr.Kind == model.Class || curr.Kind == model.Interface || curr.Kind == model.Enum)
		if resolved := mr.ResolveField(curr, seg.Name, isStatic); resolved != nil {
			return resolved
		}
		// 类型接收者上的字段解析失败（C-8）：
		// 外部类型（HttpStatus.NOT_FOUND）、外部继承实例字段（this.pageContext，父类不在符号表）、
		// 缺失的接口常量（AnnotationFilter.PLAIN）——统一构造外部字段符号并继续链，
		// 而非断链回退类型头（否则回退的类型头会被 alignElementWithRelType 虚增为构造器 CALL）。
		if curr != nil && isTypeKind(curr.Kind) {
			return model.NewExternalElement(model.Field, seg.Name, cr.gc.BuildQualifiedName(curr.QualifiedName, seg.Name))
		}
		return nil

	case SegmentMethod:
		// 方法调用：curr 为类型且处于静态上下文（类型名/静态链）时按静态解析，
		// 否则（new 表达式/变量/cast 后调用等实例接收者）按实例解析。
		// 方法引用 `Type::method` 是非绑定引用：接收者类型上的静态/实例方法都合法
		//（如 `MergedAnnotation::isPresent` 引用接口的实例方法），不走静态过滤。
		isStatic := !seg.IsMethodRef && !cr.instanceCtx &&
			(curr.Kind == model.Class || curr.Kind == model.Interface || curr.Kind == model.Enum)
		resolved := mr.ResolveMethod(curr, seg.Name, argCount, inferredTypes, isStatic)
		if resolved == nil && seg.IsMethodRef && !curr.IsFormExternal {
			// 方法引用目标不在接收者类型上时，回退到词法外层类（含当前类）查找同名方法。
			// 例：`AnnoFilter::matches` 的 matches 定义在外层类 C8FullProbe 上。
			// 仅限源码类型接收者：外部类型（如 Guava CacheBuilder）成员不在符号表内，
			// 回退外层类会把 `CacheBuilder.newBuilder()::build` 错配到本类的同名方法。
			resolved = cr.resolveMethodRefInEnclosing(seg, argCount, inferredTypes,
				(curr.Kind == model.Variable || curr.Kind == model.Field) && cr.receiverTypeUnknown(curr))
		}
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
			return cr.resolveTypeFromNameAt(seg.ASTNode, elemType)
		}
		return curr

	case SegmentClass:
		// 嵌套类访问。
		return cr.resolveTypeFromName(cr.gc.BuildQualifiedName(curr.QualifiedName, seg.Name))
	}
	return nil
}

// resolveMethodRefInEnclosing 在方法引用所在类的词法外层链上查找同名方法。
// 用于 `AnnoFilter::matches` 这类接收者类型上不存在、但定义在外层类上的方法引用
// （matches 定义于 C8FullProbe，引用点也在 C8FullProbe 内）。
func (cr *ChainResolver) resolveMethodRefInEnclosing(seg ExpressionSegment, argCount int, inferredTypes []string, allowShortName bool) *model.CodeElement {
	cls := cr.currentClass(seg.ASTNode)
	if cls == nil {
		return nil
	}
	mr := NewMemberResolver(cr.gc, cr.fc)
	for outer := cls; outer != nil; outer = cr.outerClassOf(outer) {
		if resolved := mr.ResolveMethod(outer, seg.Name, argCount, inferredTypes, false); resolved != nil {
			return resolved
		}
	}
	// 接收者类型不可知（如 lambda 形参 `input::getGroup`，声明类型为 inferred）时，
	// 按方法名做短名检索兜底，只接受源码 Method。旧实现（不拆方法引用）正是靠
	// resolveIdent 的类型名兜底命中同名方法，此处保持等价语义。
	// 仅限类型未知的接收者：已知类型（含外部类型）上查不到成员即应断链，
	// 否则 `annotations::add` 会错配到无关的源码 add 方法。
	if allowShortName {
		for _, def := range helper.PreciseResolve(cr.gc, cr.fc, seg.Name) {
			if def.Element.Kind == model.Method && !def.Element.IsFormExternal {
				return def.Element
			}
		}
	}
	return nil
}

// receiverTypeUnknown 判断变量/字段的声明类型是否不可知（lambda 形参 inferred / 空）。
func (cr *ChainResolver) receiverTypeUnknown(elem *model.CodeElement) bool {
	if elem == nil || elem.Extra == nil {
		return false
	}
	t, _ := elem.Extra.Mores[constants.VariableTypeWithQN].(string)
	if t == "" {
		t, _ = elem.Extra.Mores[constants.FieldRawType].(string)
	}
	t = helper.Clean(t)
	return t == "" || t == "inferred" || t == "var"
}

// resolveConstructor 在目标类型及其父类链上查找构造器元素（含重载消解）。
// 未命中时：
//   - Interface / Annotation 不可能有构造器（如匿名类实例化 `new Iface(){}`），返回 nil，
//     交由调用方兜底为外部符号（Balanced 下被降噪），避免产出悬空的 `Iface.Iface()`；
//   - Class / Enum 返回类型元素本身，交由 alignElementWithRelType 补构造器名段
//     （源码类但构造器未被收集时的旧语义兜底）。
func (cr *ChainResolver) resolveConstructor(typeElem *model.CodeElement, argCount int, inferredTypes []string) *model.CodeElement {
	if typeElem == nil {
		return nil
	}
	mr := NewMemberResolver(cr.gc, cr.fc)
	if elem := cr.resolveCtorOnType(mr, typeElem, typeElem.Name, argCount, inferredTypes); elem != nil {
		return elem
	}
	if typeElem.Kind == model.Interface || typeElem.Kind == model.KAnnotation {
		return nil
	}
	return typeElem
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
			if parents := helper.ResolveInheritanceParent(cr.gc, cr.fc, cls, helper.Clean(sc)); len(parents) > 0 {
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
		targetType = cr.resolveTypeStrictAt(head.ASTNode, ref)
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
		if parents := helper.ResolveInheritanceParent(cr.gc, cr.fc, elem, helper.Clean(sc)); len(parents) > 0 {
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
	// 注意：PreciseResolve 的 FindByShortName 会把字段/变量短名当类型命中返回。
	// 兜底的本意是解析“类型名”，因此必须对命中的元素做种类甄别，避免把非类型
	// 当作合法结果返回（跨方法/跨类同名变量会串扰——如 else 块变量 userInfo 被
	// 误解析到另一方法的 userInfo，见 Bug B）。
	// 合法放行的非类型命中仅两类：
	//   - Method：方法引用目标（this::foo / Class::foo）无 name 字段，findCoreChain
	//     退化为裸标识符，须借类型兜底命中同名方法；
	//   - Field：静态上下文拦截分支（见下），返回 Class kind 外部元素阻断 USE 产出。
	// 其余非类型命中（Variable 等）一律返回 NIL，交由调用方 generateContextualFallback
	// 产出外部符号（分辨率失败语义）。
	if typeEle := cr.resolveTypeFromNameAt(head.ASTNode, symbol); typeEle != nil {
		if isTypeKind(typeEle.Kind) {
			return typeEle
		}
		// 方法引用目标：this::foo / Class::foo 的 foo 是方法名，类型兜底命中 Method 合法。
		if typeEle.Kind == model.Method {
			return typeEle
		}
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
		// 其余非类型命中（Variable 等）：跨作用域串扰，拒绝。
		return nil
	}

	// 3. 外部兜底。
	return model.NewExternalElement(model.Class, symbol, cr.contextualQN(symbol))
}

// resolveVarInScope 在作用域链中查找变量/字段。
func (cr *ChainResolver) resolveVarInScope(node coreast.Node, symbol string) *model.CodeElement {
	// 委托给 SymbolResolver 的 ResolveVar（通过 gc.Resolver）。
	return cr.gc.Resolver().ResolveVar(cr.gc, cr.fc, node, "", symbol)
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
// node 为使用点（D-1 消歧用）：声明类型为同文件同名嵌套类型时，按引用点外层类作用域选择。
func (cr *ChainResolver) receiverContainer(elem *model.CodeElement, node coreast.Node) *model.CodeElement {
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

	container := cr.resolveTypeStrictAt(node, typeQN)
	// 声明类型可能只在「声明该变量/字段的文件」可见（如父类文件内定义的嵌套类型），
	// 使用点所在文件既无 import 也无定义，按使用点解析会退化成外部裸名，
	// 后续成员段全部断裂（`httpClient.events.clear()` 退化为 target=httpClient）。
	// 此时改用元素自身 Path 对应的 FileContext 再解析一次。
	if (container == nil || container.IsFormExternal) && elem.Path != "" {
		if declFC, ok := cr.gc.FileContext(elem.Path); ok && declFC != cr.fc {
			if resolved := helper.PreciseResolve(cr.gc, declFC, typeQN); len(resolved) > 0 {
				if isTypeKind(resolved[0].Element.Kind) {
					container = resolved[0].Element
				}
			}
		}
	}
	if container == nil || !isTypeKind(container.Kind) {
		return nil
	}
	return container
}

// varGenericElem 从变量/字段的声明类型文本中提取泛型实参（C<X> 的 X）。
// 例：Supplier<SpringValidatorAdapter> -> SpringValidatorAdapter；List<Order> -> Order。
// 仅提取单个顶层泛型实参（嵌套泛型取最内层首个实参）；无泛型/类型为空返回空串。
// 用途（C-3 #95）：外部容器（如 java.util.function.Supplier）上的方法段（如 .get()）
// 解析为外部方法且无返回类型元数据时，用泛型实参 X 作为方法返回类型，使后续段
// （.getArgumentsForConstraint()）能在 X 上继续解析。仅依赖源码内类型文本，不建外部
// 类型签名映射表。
func (cr *ChainResolver) varGenericElem(elem *model.CodeElement) string {
	if elem == nil || elem.Extra == nil {
		return ""
	}
	typeText, _ := elem.Extra.Mores[constants.VariableTypeWithQN].(string)
	if typeText == "" {
		typeText, _ = elem.Extra.Mores[constants.FieldRawType].(string)
	}
	if !strings.Contains(typeText, "<") {
		return ""
	}
	start, end := strings.Index(typeText, "<"), strings.LastIndex(typeText, ">")
	if start == -1 || end == -1 || start >= end {
		return ""
	}
	inner := typeText[start+1 : end]
	if strings.Contains(inner, "<") {
		inner = strings.Split(inner, ",")[0]
	}
	return helper.Clean(inner)
}

// singleTypeArg 判断变量/字段的声明类型是否为单类型参数容器（C<X>，无逗号分隔的多参数）。
// 例：Supplier<X>/List<X>/AtomicReference<X> -> true；Map<K,V>/ConverterFactory<S,T> -> false。
// 用途（C-3 #95 收窄）：多类型参数容器的 get 方法（如 ConverterFactory<S,T>.getConverter()）
// 返回类型不确定（非首个类型参数），不可用首实参做传播，否则后续段会错误解析到首实参上。
func (cr *ChainResolver) singleTypeArg(elem *model.CodeElement) bool {
	if elem == nil || elem.Extra == nil {
		return false
	}
	typeText, _ := elem.Extra.Mores[constants.VariableTypeWithQN].(string)
	if typeText == "" {
		typeText, _ = elem.Extra.Mores[constants.FieldRawType].(string)
	}
	if !strings.Contains(typeText, "<") {
		return false
	}
	start, end := strings.Index(typeText, "<"), strings.LastIndex(typeText, ">")
	if start == -1 || end == -1 || start >= end {
		return false
	}
	return !strings.Contains(typeText[start+1:end], ",")
}

// classLiteralContainer 返回 java.lang.Class 外部元素，作为 Xxx.class 带段链的接收者容器。
// 成员方法（getCanonicalName/getName/getSimpleName 等）均为 JDK 方法，解析为外部符号。
func (cr *ChainResolver) classLiteralContainer() *model.CodeElement {
	return model.NewExternalElement(model.Class, "Class", "java.lang.Class")
}

// methodReturnContainer 依据方法的返回类型解析接收者容器（类/接口/枚举）。
// 用于链式调用：方法段的后续段在返回类型上解析成员（如 new Builder().name().age()）。
// node 为后续段使用点（D-1 消歧用）：返回类型为同文件同名嵌套类型时按引用点外层类选择。
func (cr *ChainResolver) methodReturnContainer(elem *model.CodeElement, node coreast.Node) *model.CodeElement {
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

	container := cr.resolveTypeStrictAt(node, retQN)
	if container == nil || !isTypeKind(container.Kind) {
		return nil
	}
	return container
}

// resolveTypeStrictAt 在类型位置解析类型名，只接受类型类命中。
// 与 resolveTypeFromNameAt 的区别：同名非类型定义（枚举常量/字段/变量遮蔽类型名，
// 如 `enum CacheType { ConcurrentHashMap }` 内的 `new ConcurrentHashMap<>()`）不得作为
// 类型结果返回，须按 import / 同包 / 全局 QN 重新解析，仍未命中则产外部类型符号。
func (cr *ChainResolver) resolveTypeStrictAt(node coreast.Node, name string) *model.CodeElement {
	// 已是完整 QN 的声明类型（如局部类参数 `...Test.method().Target`）直接命中：
	// helper.Clean 会把 QN 在首个 "(" 处截断（方法 QN 自带参数列表），
	// 先按原样精确查找可避免嵌套类的声明类型解析失败。
	if name != "" {
		if def, ok := cr.gc.FindByQualifiedName(name); ok && isTypeKind(def.Element.Kind) {
			return def.Element
		}
	}
	clean := helper.Clean(name)
	if clean == "" {
		return nil
	}
	// 基本类型产 PRIMITIVE 专用元素（保留类型语义）。
	if helper.IsPrimitiveType(clean) {
		return &model.CodeElement{Name: clean, QualifiedName: clean, Kind: model.Primitive, IsFormExternal: true}
	}
	// 先按符号表解析：单大写字母既可能是泛型类型变量，也可能是真实类名
	//（如测试内的局部类 X），只有确实未命中时才按类型变量丢弃。
	if elem := cr.resolveTypeFromNameAt(node, clean); elem != nil && isTypeKind(elem.Kind) {
		return elem
	}
	if helper.IsTypeVariable(clean) {
		return nil
	}

	// 单类型导入 / 通配符导入 / 同包 / 全局 QN 逐级重解析（仅接受类型类）。
	if imps, ok := cr.fc.Imports()[clean]; ok {
		for _, imp := range imps {
			if imp.BoolAttr(constants.ImportAttrIsWildcard) {
				basePath := strings.TrimSuffix(imp.RawImportPath, "*")
				if def, ok := cr.gc.FindByQualifiedName(basePath + clean); ok && isTypeKind(def.Element.Kind) {
					return def.Element
				}
				continue
			}
			if def, ok := cr.gc.FindByQualifiedName(imp.RawImportPath); ok && isTypeKind(def.Element.Kind) {
				return def.Element
			}
		}
	}
	if pkgQN := cr.gc.BuildQualifiedName(helper.PackageNameOf(cr.fc), clean); pkgQN != clean {
		if def, ok := cr.gc.FindByQualifiedName(pkgQN); ok && isTypeKind(def.Element.Kind) {
			return def.Element
		}
	}
	if def, ok := cr.gc.FindByQualifiedName(clean); ok && isTypeKind(def.Element.Kind) {
		return def.Element
	}

	// 外部类型兜底：有 import 用 import 包前缀，否则归 <external>（与 ResolveType 口径一致）。
	qn, pkg := clean, "<external>"
	if imps, ok := cr.fc.Imports()[clean]; ok {
		for _, imp := range imps {
			if imp.BoolAttr(constants.ImportAttrIsWildcard) {
				continue
			}
			qn, pkg = imp.RawImportPath, helper.PkgOf(imp.RawImportPath)
			break
		}
	}
	return &model.CodeElement{Name: clean, QualifiedName: qn, NamespaceQN: pkg, Kind: model.Class, IsFormExternal: true}
}

// resolveTypeFromName 将类型名解析为类型元素。
func (cr *ChainResolver) resolveTypeFromName(name string) *model.CodeElement {
	name = helper.Clean(name)
	if name == "" {
		return nil
	}
	return cr.gc.Resolver().ResolveType(cr.gc, cr.fc, name, model.Class)
}

// resolveTypeFromNameAt 将类型名解析为类型元素，node 为引用点（D-1 消歧用）。
// 同一文件存在多个同名嵌套类型时（如 DataBufferUtils 的 Read/WriteCompletionHandler.Attachment），
// 依据引用点所在的外层类作用域优先选择，避免按文件收集顺序错配到先定义者。
func (cr *ChainResolver) resolveTypeFromNameAt(node coreast.Node, name string) *model.CodeElement {
	name = helper.Clean(name)
	if name == "" {
		return nil
	}
	if node != nil && !strings.Contains(name, ".") {
		if defs, ok := cr.fc.FindByShortName(name); ok {
			if scoped := helper.ResolveInScope(cr.gc, cr.fc, node, defs); len(scoped) > 0 {
				// 只接受类型类命中：同文件的枚举常量/字段/变量可能与被引用类型同名
				// （如 enum CacheType { ConcurrentHashMap } 内 `new ConcurrentHashMap<>()`），
				// 直接返回非类型会把类型引用错解析成同名成员。
				if isTypeKind(scoped[0].Element.Kind) {
					return scoped[0].Element
				}
			}
		}
	}
	return cr.gc.Resolver().ResolveType(cr.gc, cr.fc, name, model.Class)
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
	if imps, ok := cr.fc.Imports()[symbol]; ok && len(imps) > 0 {
		return imps[0].RawImportPath
	}
	return symbol
}
