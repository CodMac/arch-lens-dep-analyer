package java

import (
	"slices"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type SymbolResolver struct{}

func NewJavaSymbolResolver() *SymbolResolver {
	return &SymbolResolver{}
}

// =============================================================================
// 接口实现
// =============================================================================

// BuildQualifiedName 根据父节点和当前名构建 QN
func (j *SymbolResolver) BuildQualifiedName(parentQN, name string) string {
	if parentQN == "" || parentQN == "." {
		return name
	}
	return parentQN + "." + name
}

// RegisterPackage 注册包
func (j *SymbolResolver) RegisterPackage(gc *core.GlobalContext, packageName string) {
	parts := strings.Split(packageName, ".")
	var current []string
	for _, part := range parts {
		current = append(current, part)
		pkgQN := strings.Join(current, ".")
		if _, ok := gc.FindByQualifiedName(pkgQN); !ok {
			entry := core.DefinitionEntry{
				Element: &model.CodeElement{Kind: model.Package, Name: part, QualifiedName: pkgQN, IsFormSource: true},
			}
			gc.AddDefinition(&entry)
		}
	}
}

// IsPrimitive 是否为基础类型
func (j *SymbolResolver) IsPrimitive(typeName string) bool {
	switch typeName {
	case "int", "long", "short", "byte", "char", "boolean", "float", "double":
		return true
	}
	return false
}

// ResolveVar 处理变量查找，支持本地作用域回溯和类成员继承查找
func (j *SymbolResolver) ResolveVar(gc *core.GlobalContext, fc *core.FileContext, node *sitter.Node, receiver string, symbol string) *model.CodeElement {
	receiver = helper.Clean(receiver)
	symbol = helper.Clean(symbol)

	isStatic := false

	if receiver != "" {
		// 场景 A: this 或 super
		if receiver == "this" || receiver == "super" {
			container := helper.GetBestElement(fc, node, []model.ElementKind{model.Class, model.AnonymousClass})
			if container == nil {
				return nil
			}
			isStatic = slices.Contains(container.Extra.Modifiers, "static")

			startEntry := container
			if receiver == "super" {
				return j.resolveFromInheritance(gc, fc, container, symbol, isStatic, container)
			}

			return j.resolveInScopeHierarchy(gc, fc, startEntry.QualifiedName, symbol, isStatic, container)
		}

		// 场景 B: 尝试解析为类名 (静态访问)
		// 先清理 receiver (如 List<String> -> List)
		if entries := helper.PreciseResolve(gc, fc, receiver); len(entries) > 0 {
			// 如果解析结果是类/接口，则按静态字段查找
			receiverEle := entries[0].Element
			if receiverEle.Kind == model.Class || receiverEle.Kind == model.Interface || receiverEle.Kind == model.AnonymousClass {
				return j.resolveInScopeHierarchy(gc, fc, receiverEle.QualifiedName, symbol, true, receiverEle)
			}
		}

		// 场景 C: 跨对象访问 (data.age)
		// 先解析 receiver 变量本身拿到它的类型
		receiverEle := j.ResolveVar(gc, fc, node, "", receiver)
		if receiverEle != nil && receiverEle.Extra != nil {
			if typeQN, ok := receiverEle.Extra.Mores[constants.VariableTypeWithQN].(string); ok {
				if entries := helper.PreciseResolve(gc, fc, typeQN); len(entries) > 0 {
					receiverTypeEle := entries[0].Element
					if receiverTypeEle.Kind == model.Class || receiverTypeEle.Kind == model.Interface || receiverTypeEle.Kind == model.AnonymousClass {
						return j.resolveInScopeHierarchy(gc, fc, receiverTypeEle.QualifiedName, symbol, false, receiverEle)
					}
				}
			}
		}
	}

	// 无 receiver：按原有作用域链查找
	container := helper.GetBestElement(fc, node, []model.ElementKind{model.Method, model.Class, model.ScopeBlock})
	if container == nil {
		return nil
	}
	isStatic = slices.Contains(container.Extra.Modifiers, "static")
	return j.resolveInScopeHierarchy(gc, fc, container.QualifiedName, symbol, isStatic, container)
}

// ResolveFunc 处理方法查找：容器定位 -> 继承链搜索 -> 重载消解
func (j *SymbolResolver) ResolveFunc(gc *core.GlobalContext, fc *core.FileContext, node *sitter.Node, receiver string, symbol string) *model.CodeElement {
	receiver = helper.Clean(receiver)
	symbol = helper.Clean(symbol)

	var container *model.CodeElement
	var isStaticCall bool

	// 1. 确定搜索的起始容器
	if receiver != "" {
		// 场景 A: 静态调用 (类名.method)
		if entries := helper.PreciseResolve(gc, fc, receiver); len(entries) > 0 {
			first := entries[0].Element
			switch first.Kind {
			case model.Class:
				container = first
				isStaticCall = true
			case model.Interface:
				container = first
			case model.Enum:
				container = first
			}
		}

		// 场景 B: this/super 调用
		if container == nil && (receiver == "this" || receiver == "super") {
			container = helper.GetBestElement(fc, node, []model.ElementKind{model.Class, model.AnonymousClass})
			if receiver == "super" && container != nil {
				// 如果是 super，容器直接指向父类
				if sc, ok := container.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
					if parents := helper.PreciseResolve(gc, fc, helper.Clean(sc)); len(parents) > 0 {
						container = parents[0].Element
					}
				}
			}
		}

		// 场景 C: 实例变量调用 (obj.method)
		if container == nil {
			if recvVar := j.ResolveVar(gc, fc, node, "", receiver); recvVar != nil {
				// 利用 Binder 补全的 QN 定位类
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

	// 场景 D: 无 receiver，从当前代码位置寻找最近的类
	if container == nil {
		container = helper.GetBestElement(fc, node, []model.ElementKind{model.Class, model.AnonymousClass})
	}

	if container == nil {
		return &model.CodeElement{Name: symbol, Kind: model.Method, IsFormExternal: true}
	}

	// 2. 准备调用处的实参信息 (用于重载匹配)
	argCount := 0
	var inferredArgTypes []string
	if node != nil {
		if invNode := j.findInvocationNode(node); invNode != nil {
			if args := invNode.ChildByFieldName("arguments"); args != nil {
				argCount = int(args.NamedChildCount())
				inferredArgTypes = j.inferArgumentTypes(args, fc)
			}
		}
	}

	// 3. 沿继承链向上搜索
	result := j.searchMethodInHierarchy(gc, fc, container, symbol, argCount, inferredArgTypes, isStaticCall, container)
	if result != nil {
		return result
	}

	// 4. 最终找不到，返回外部符号
	return &model.CodeElement{
		Name:           symbol,
		QualifiedName:  symbol, // 或者是 container.QN + "." + symbol
		Kind:           model.Method,
		IsFormExternal: true,
	}
}

// ResolveType 解析结构体符号(Package、Class、Interface、AnonymousClass、Enum......), 如果上下文没找到，则返回kind类型的外部实体
func (j *SymbolResolver) ResolveType(gc *core.GlobalContext, fc *core.FileContext, symbol string, kind model.ElementKind) *model.CodeElement {
	symbol = helper.Clean(symbol)

	if entries := helper.PreciseResolve(gc, fc, symbol); len(entries) > 0 {
		return entries[0].Element
	}

	// 找不到明确实体，则尝试根据Imports升级符号，返回一个外部实体
	qualifiedName := symbol
	if imps, ok := fc.Imports[symbol]; ok && len(imps) > 0 {
		qualifiedName = imps[0].RawImportPath
	}
	return &model.CodeElement{Name: symbol, QualifiedName: qualifiedName, Kind: kind, IsFormExternal: true}
}

// ResolveAction 统一的动作解析入口，根据目标节点和上下文节点提取文本和接收者，然后调用对应的解析方法
func (j *SymbolResolver) ResolveAction(gc *core.GlobalContext, fc *core.FileContext, targetNode *sitter.Node, ctxNode *sitter.Node, relType model.DependencyType) *model.CodeElement {
	src := *fc.SourceBytes
	symbol := targetNode.Utf8Text(src)

	switch relType {
	case model.Call:
		receiverText := j.extractReceiverFromCallCtx(ctxNode, src)
		return j.ResolveFunc(gc, fc, targetNode, receiverText, symbol)
	case model.Assign, model.Use:
		receiverText := j.extractReceiverFromFieldAccess(fc, targetNode, src)
		return j.ResolveVar(gc, fc, targetNode, receiverText, symbol)
	default:
		return j.ResolveType(gc, fc, symbol, model.Class)
	}
}

// =============================================================================
// 查找逻辑
// =============================================================================

// resolveInScopeHierarchy 递归向上查找容器及继承链
func (j *SymbolResolver) resolveInScopeHierarchy(gc *core.GlobalContext, fc *core.FileContext, previousQN, symbol string, isStatic bool, container *model.CodeElement) *model.CodeElement {
	if previousQN == "" {
		return nil
	}

	// 1. 尝试在当前层级直接匹配
	targetQN := j.BuildQualifiedName(previousQN, symbol)
	if entry, ok := gc.FindByQualifiedName(targetQN); ok {
		if j.checkVisibility(gc, fc, container, entry) {
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

	// 2. 如果是类/接口，递归查找其继承链 (extends/implements)
	previousEleKind := previousEntry.Element.Kind
	if previousEleKind == model.Class || previousEleKind == model.Interface || previousEleKind == model.AnonymousClass {
		if inherited := j.resolveFromInheritance(gc, fc, previousEntry.Element, symbol, isStatic, container); inherited != nil {
			return inherited
		}
	}

	// 3. 递归到上一级 Lexical Scope
	return j.resolveInScopeHierarchy(gc, fc, previousEntry.ParentQN, symbol, isStatic, container)
}

// resolveFromInheritance 处理继承树查找
func (j *SymbolResolver) resolveFromInheritance(gc *core.GlobalContext, fc *core.FileContext, elem *model.CodeElement, symbol string, isStatic bool, sourceElem *model.CodeElement) *model.CodeElement {
	if elem.Extra == nil {
		return nil
	}

	var superTargets []string
	if sc, ok := elem.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
		superTargets = append(superTargets, sc)
	}
	if itfs, ok := elem.Extra.Mores[constants.ClassImplementedInterfaces].([]string); ok {
		superTargets = append(superTargets, itfs...)
	}

	for _, rawSuperName := range superTargets {
		cleanSuperName := strings.Split(rawSuperName, "<")[0]
		parentEntries := helper.PreciseResolve(gc, fc, cleanSuperName)

		if len(parentEntries) > 0 {
			parentElem := parentEntries[0].Element
			targetQN := j.BuildQualifiedName(parentElem.QualifiedName, symbol)

			if fieldEntry, ok := gc.FindByQualifiedName(targetQN); ok {
				if j.checkVisibility(gc, fc, sourceElem, fieldEntry) {
					if !isStatic || slices.Contains(fieldEntry.Element.Extra.Modifiers, "static") {
						return fieldEntry.Element
					}
				}
			}
			// 深度优先递归父类的父类
			if found := j.resolveFromInheritance(gc, fc, parentElem, symbol, isStatic, sourceElem); found != nil {
				return found
			}
		}
	}
	return nil
}

// searchMethodInHierarchy 递归搜索当前类及父类/接口
func (j *SymbolResolver) searchMethodInHierarchy(gc *core.GlobalContext, fc *core.FileContext, currContainer *model.CodeElement, symbol string, argCount int, inferredTypes []string, isStaticCall bool, source *model.CodeElement) *model.CodeElement {
	if currContainer == nil {
		return nil
	}

	// A. 查找当前容器内所有同名方法
	targetPrefix := currContainer.QualifiedName + "." + symbol
	var candidates []*core.DefinitionEntry

	// 从全局上下文获取所有同名的 方法QN
	if entries, ok := gc.FindMethodByNoParamsQN(targetPrefix); ok {
		for _, e := range entries {
			if e.Element.Kind != model.Method {
				continue
			}
			// 静态检查：如果是静态调用，只能看静态方法
			if isStaticCall && !slices.Contains(e.Element.Extra.Modifiers, "static") {
				continue
			}
			// 可见性检查
			if j.checkVisibility(gc, fc, source, e) {
				candidates = append(candidates, e)
			}
		}
	}

	// B. 如果有同名候选，进行重载匹配
	if len(candidates) > 0 {
		return j.pickBestOverloadEnhanced(candidates, argCount, inferredTypes)
	}

	// C. 当前类没找到，递归查找父类 (Extends)
	if sc, ok := currContainer.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
		if parents := helper.PreciseResolve(gc, fc, helper.Clean(sc)); len(parents) > 0 {
			if res := j.searchMethodInHierarchy(gc, fc, parents[0].Element, symbol, argCount, inferredTypes, isStaticCall, source); res != nil {
				return res
			}
		}
	}

	// D. 递归查找接口 (Implements)
	if itfs, ok := currContainer.Extra.Mores[constants.ClassImplementedInterfaces].([]string); ok {
		for _, itf := range itfs {
			if parents := helper.PreciseResolve(gc, fc, helper.Clean(itf)); len(parents) > 0 {
				if res := j.searchMethodInHierarchy(gc, fc, parents[0].Element, symbol, argCount, inferredTypes, isStaticCall, source); res != nil {
					return res
				}
			}
		}
	}

	return nil
}

// =============================================================================
// 4. 重载与类型匹配辅助 (Overload & Type Inference)
// =============================================================================

// pickBestOverloadEnhanced 结合参数数量和启发式类型匹配选择最优重载
func (j *SymbolResolver) pickBestOverloadEnhanced(entries []*core.DefinitionEntry, argCount int, inferredTypes []string) *model.CodeElement {
	var bestMatch *model.CodeElement
	maxScore := -1

	for _, entry := range entries {
		definedParamCount := 0
		currentScore := 0

		// 获取 Binder 补全后的参数 QN 列表, 格式为 ["String name", "int age"]
		params, ok := entry.Element.Extra.Mores[constants.MethodParametersWithQN].([]string)
		if ok {
			definedParamCount = len(params)
		}

		// 1. 严格匹配参数数量 (基础分)
		if definedParamCount == argCount {
			currentScore += 100

			// 2. 匹配参数类型
			for i := 0; i < argCount; i++ {
				definedTypeQN := helper.Clean(params[i])
				inferredType := inferredTypes[i] // 实参推断出的类型（可能是短名或 QN）

				if inferredType == "unknown" || inferredType == "null" {
					currentScore += 10 // 模糊匹配给个保底分
					continue
				}

				// 因为有了 Binder，我们可以做更精准的对比
				if definedTypeQN == inferredType || strings.HasSuffix(definedTypeQN, "."+inferredType) {
					currentScore += 50
				}
			}
		}

		if currentScore > maxScore {
			maxScore = currentScore
			bestMatch = entry.Element
		}
	}

	if bestMatch != nil {
		return bestMatch
	}
	return entries[0].Element // 兜底返回第一个
}

// inferArgumentTypes 尝试从实参 AST 节点推断大致类型
func (j *SymbolResolver) inferArgumentTypes(argsNode *sitter.Node, fc *core.FileContext) []string {
	var types []string
	src := *fc.SourceBytes

	for i := 0; i < int(argsNode.NamedChildCount()); i++ {
		arg := argsNode.NamedChild(uint(i))
		kind := arg.Kind()

		switch kind {
		case "string_literal":
			types = append(types, "String")
		case "decimal_integer_literal", "hex_integer_literal":
			types = append(types, "int")
		case "decimal_floating_point_literal":
			types = append(types, "double")
		case "true", "false", "boolean_type":
			types = append(types, "boolean")
		case "null_literal":
			types = append(types, "null")
		case "object_creation_expression", "cast_expression":
			if typeNode := arg.ChildByFieldName("type"); typeNode != nil {
				types = append(types, helper.GetNodeContent(typeNode, src))
			} else {
				types = append(types, "unknown")
			}
		case "array_creation_expression":
			if typeNode := arg.ChildByFieldName("type"); typeNode != nil {
				types = append(types, helper.GetNodeContent(typeNode, src)+"[]")
			} else {
				types = append(types, "unknown")
			}
		default:
			types = append(types, "unknown")
		}
	}
	return types
}

// =============================================================================
// 5. 校验与底层工具 (Utilities)
// =============================================================================

func (j *SymbolResolver) checkVisibility(gc *core.GlobalContext, fc *core.FileContext, container *model.CodeElement, target *core.DefinitionEntry) bool {
	// 1. 局部变量/形参/Lambda参数无限制
	if target.Element.Kind == model.Variable {
		return true
	}

	// 2. 检查是否属于同一个顶层类 (处理内部类、匿名类)
	containerOutermost := helper.GetOutermostClassQN(container.QualifiedName)
	targetOutermost := helper.GetOutermostClassQN(target.Element.QualifiedName)
	if containerOutermost != "" && containerOutermost == targetOutermost {
		return true
	}

	// 3. 显式修饰符判断
	if target.Element.Extra == nil || target.Element.Extra.Modifiers == nil {
		return false
	}
	mods := target.Element.Extra.Modifiers
	if slices.Contains(mods, "public") {
		return true
	}

	// 4. 包级私有 (Default/Package-Private) 判定
	// 注意：getPackageFromQN 应该确保拿到真正的 Java Package 名
	targetPkg := helper.GetRealPackage(gc, target.Element)
	if targetPkg == fc.PackageName {
		return true
	}

	// 5. Protected: 检查子类关系
	if slices.Contains(mods, "protected") {
		sourceClass := helper.GetOwnerClassQN(gc, container)
		return j.isSubClassOf(gc, fc, sourceClass, target.ParentQN)
	}

	return false
}

func (j *SymbolResolver) determinePreciseContainer(fc *core.FileContext, node *sitter.Node, kinds []model.ElementKind) *model.CodeElement {
	if node == nil {
		return nil
	}
	var best *model.CodeElement
	var minSize uint32 = 0xFFFFFFFF
	row := int(node.StartPosition().Row + 1)
	for _, entry := range fc.Definitions {
		if slices.Contains(kinds, entry.Element.Kind) {
			if row >= entry.Element.Location.StartLine && row <= entry.Element.Location.EndLine {
				size := uint32(entry.Element.Location.EndLine - entry.Element.Location.StartLine)
				if size < minSize {
					minSize, best = size, entry.Element
				}
			}
		}
	}
	return best
}

func (j *SymbolResolver) isSubClassOf(gc *core.GlobalContext, fc *core.FileContext, sub, super string) bool {
	if sub == "" || super == "" || sub == super {
		return sub == super
	}
	entry, ok := gc.FindByQualifiedName(sub)
	if !ok || entry.Element.Extra == nil {
		return false
	}
	if sc, ok := entry.Element.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
		parents := helper.PreciseResolve(gc, fc, strings.Split(sc, "<")[0])
		for _, p := range parents {
			if p.Element.QualifiedName == super || j.isSubClassOf(gc, fc, p.Element.QualifiedName, super) {
				return true
			}
		}
	}
	return false
}

func (j *SymbolResolver) findInvocationNode(n *sitter.Node) *sitter.Node {
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

// extractReceiverFromCallCtx 从调用上下文中提取接收者对象
func (j *SymbolResolver) extractReceiverFromCallCtx(ctxNode *sitter.Node, src []byte) string {
	if ctxNode == nil {
		return ""
	}
	if obj := ctxNode.ChildByFieldName("object"); obj != nil {
		return obj.Utf8Text(src)
	}
	return ""
}

// extractReceiverFromFieldAccess 从字段访问上下文中提取接收者对象
func (j *SymbolResolver) extractReceiverFromFieldAccess(fc *core.FileContext, targetNode *sitter.Node, src []byte) string {
	if targetNode == nil {
		return ""
	}

	parent := targetNode.Parent()
	if parent != nil && parent.Kind() == "field_access" {
		if obj := parent.ChildByFieldName("object"); obj != nil {
			receiverText := obj.Utf8Text(src)
			if obj.Id() != targetNode.Id() {
				return receiverText
			}
		}
	}
	return ""
}
