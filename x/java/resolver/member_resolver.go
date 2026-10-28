package resolver

import (
	"slices"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

// MemberResolver 提供方法/字段成员解析：候选收集（含继承回溯）-> 重载打分。
type MemberResolver struct {
	gc *core.GlobalContext
	fc *core.FileContext
}

// NewMemberResolver 创建成员解析器。
func NewMemberResolver(gc *core.GlobalContext, fc *core.FileContext) *MemberResolver {
	return &MemberResolver{gc: gc, fc: fc}
}

// ResolveMethod 解析容器上的方法成员。
// container 为当前符号（类/接口/枚举），symbol 为方法名，argCount/argTypes 用于重载消解。
func (mr *MemberResolver) ResolveMethod(container *model.CodeElement, symbol string, argCount int, inferredTypes []string, isStaticCall bool) *model.CodeElement {
	if container == nil {
		return nil
	}

	// 收集候选（当前容器 + 继承链回溯）。
	candidates := mr.collectMethodCandidates(container, symbol, isStaticCall)
	if len(candidates) == 0 {
		return nil
	}

	return mr.pickBestOverload(candidates, argCount, inferredTypes)
}

// collectMethodCandidates 收集容器内同名方法候选（沿继承链回溯去重）。
func (mr *MemberResolver) collectMethodCandidates(container *model.CodeElement, symbol string, isStaticCall bool) []*core.DefinitionEntry {
	var out []*core.DefinitionEntry
	seen := make(map[string]bool)

	var visit func(elem *model.CodeElement)
	visit = func(elem *model.CodeElement) {
		if elem == nil || elem.Extra == nil {
			return
		}

		// 在当前容器查找同名方法。
		prefix := mr.gc.BuildQualifiedName(elem.QualifiedName, symbol)
		if entries, ok := mr.gc.FindMethodByNoParamsQN(prefix); ok {
			for _, e := range entries {
				if e.Element.Kind != model.Method {
					continue
				}
				if isStaticCall && !slices.Contains(e.Element.Extra.Modifiers, "static") && !mr.isImplicitEnumMethod(e.Element) {
					continue
				}
				if !seen[e.Element.QualifiedName] {
					seen[e.Element.QualifiedName] = true
					out = append(out, e)
				}
			}
		}

		// 递归父类。
		if sc, ok := elem.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
			if parents := helper.PreciseResolve(mr.gc, mr.fc, helper.Clean(sc)); len(parents) > 0 {
				visit(parents[0].Element)
			}
		}

		// 递归接口。
		if itfs, ok := elem.Extra.Mores[constants.ClassImplementedInterfaces].([]string); ok {
			for _, itf := range itfs {
				if parents := helper.PreciseResolve(mr.gc, mr.fc, helper.Clean(itf)); len(parents) > 0 {
					visit(parents[0].Element)
				}
			}
		}
	}

	visit(container)
	return out
}

// ResolveField 解析容器上的字段/枚举常量成员（含继承回溯、静态上下文校验）。
func (mr *MemberResolver) ResolveField(container *model.CodeElement, symbol string, isStatic bool) *model.CodeElement {
	if container == nil {
		return nil
	}

	// 直接命中。
	targetQN := mr.gc.BuildQualifiedName(container.QualifiedName, symbol)
	if entry, ok := mr.gc.FindByQualifiedName(targetQN); ok {
		if mr.checkVisibility(container, entry) {
			isIllegalStatic := isStatic && entry.Element.Kind == model.Field && !slices.Contains(entry.Element.Extra.Modifiers, "static")
			if !isIllegalStatic {
				return entry.Element
			}
		}
	}

	// 继承回溯。
	return mr.resolveFromInheritance(container, symbol, isStatic)
}

// resolveFromInheritance 沿继承树查找字段。
func (mr *MemberResolver) resolveFromInheritance(elem *model.CodeElement, symbol string, isStatic bool) *model.CodeElement {
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
		if parents := helper.PreciseResolve(mr.gc, mr.fc, cleanSuper); len(parents) > 0 {
			parent := parents[0].Element
			targetQN := mr.gc.BuildQualifiedName(parent.QualifiedName, symbol)
			if entry, ok := mr.gc.FindByQualifiedName(targetQN); ok {
				if mr.checkVisibility(elem, entry) {
					if !isStatic || slices.Contains(entry.Element.Extra.Modifiers, "static") {
						return entry.Element
					}
				}
			}
			if found := mr.resolveFromInheritance(parent, symbol, isStatic); found != nil {
				return found
			}
		}
	}
	return nil
}

// isImplicitEnumMethod 判断是否为编译器为枚举生成的隐式方法（values/valueOf）。
// 此类方法由 DesugarEnumMethods 合成（IsFormSugar:true, MethodIsImplicit:true），
// 在符号表中不带 static 修饰符，但语义上等价静态调用，静态调用过滤须放行。
func (mr *MemberResolver) isImplicitEnumMethod(elem *model.CodeElement) bool {
	if elem == nil || elem.Extra == nil {
		return false
	}
	implicit, _ := elem.Extra.Mores[constants.MethodIsImplicit].(bool)
	return implicit
}

// pickBestOverload 重载打分：参数数量 -> 类型精确匹配 -> 子类/接口 -> 装箱。
func (mr *MemberResolver) pickBestOverload(entries []*core.DefinitionEntry, argCount int, inferredTypes []string) *model.CodeElement {
	var best *model.CodeElement
	maxScore := -1

	for _, entry := range entries {
		elem := entry.Element
		currentScore := 0
		params, _ := elem.Extra.Mores[constants.MethodParametersWithQN].([]string)
		definedCount := len(params)

		// 变长参数：最后一个形参以 ... 结尾（MethodParametersWithQN 保留原始文本；
		// 不能对 Clean 后的值判断，Clean 会剥掉尾部 "..."）。
		isVarargs := definedCount > 0 && strings.HasSuffix(params[definedCount-1], "...")

		// 1. 参数数量匹配：
		//    - 数量相等：强匹配
		//    - 恰好多 1：弱匹配（可能是缺省/近似）
		//    - 变长参数且实参数 >= 固定部分：可接受
		//    - 实参过多且非变长：不适用，跳过（避免单参重载抢走变长调用）。
		switch {
		case definedCount == argCount:
			currentScore += 100
		case definedCount == argCount+1:
			currentScore += 50
		case isVarargs && argCount >= definedCount-1:
			currentScore += 60
		default:
			continue
		}

		// 2. 类型精确匹配。
		for i := 0; i < argCount && i < definedCount; i++ {
			if inferredTypes == nil || i >= len(inferredTypes) {
				break
			}
			definedType := helper.Clean(params[i])
			inferred := inferredTypes[i]
			if inferred == "unknown" || inferred == "null" {
				currentScore += 10
				continue
			}
			if definedType == inferred || strings.HasSuffix(definedType, "."+inferred) {
				currentScore += 50
			} else if mr.isAssignable(definedType, inferred) {
				currentScore += 40
			} else if mr.isBoxingMatch(definedType, inferred) {
				currentScore += 30
			}
		}

		if currentScore > maxScore {
			maxScore = currentScore
			best = elem
		}
	}

	if best != nil {
		return best
	}
	return entries[0].Element
}

// isAssignable 判断 inferred 是否为 defined 的子类/实现类。
func (mr *MemberResolver) isAssignable(definedType, inferred string) bool {
	if defs, ok := mr.gc.FindByQualifiedName(definedType); ok {
		return helper.IsSubClassOf(mr.gc, mr.fc, defs.Element.QualifiedName, inferred) ||
			helper.IsSubClassOf(mr.gc, mr.fc, inferred, defs.Element.QualifiedName)
	}
	return false
}

// isBoxingMatch 基本类型装箱匹配（int<->Integer 等）。
func (mr *MemberResolver) isBoxingMatch(definedType, inferred string) bool {
	boxed := map[string]string{
		"int": "Integer", "long": "Long", "double": "Double",
		"float": "Float", "boolean": "Boolean", "char": "Character",
		"byte": "Byte", "short": "Short",
	}
	if b, ok := boxed[definedType]; ok {
		return b == inferred || strings.HasSuffix(inferred, "."+b)
	}
	for dt, b := range boxed {
		if b == definedType || strings.HasSuffix(definedType, "."+b) {
			return dt == inferred
		}
	}
	return false
}

// checkVisibility 可见性检查（同顶层类 / public / 同包 / protected 子类）。
func (mr *MemberResolver) checkVisibility(container *model.CodeElement, target *core.DefinitionEntry) bool {
	// 局部变量/形参无限制。
	if target.Element.Kind == model.Variable {
		return true
	}

	// 同顶层类。
	containerOuter := helper.GetOutermostClassQN(container.QualifiedName)
	targetOuter := helper.GetOutermostClassQN(target.Element.QualifiedName)
	if containerOuter != "" && containerOuter == targetOuter {
		return true
	}

	if target.Element.Extra == nil {
		return false
	}
	mods := target.Element.Extra.Modifiers
	if slices.Contains(mods, "public") {
		return true
	}

	// 同包。
	targetPkg := helper.GetRealPackage(mr.gc, target.Element)
	if targetPkg == mr.fc.PackageName {
		return true
	}

	// protected 子类。
	if slices.Contains(mods, "protected") {
		sourceClass := helper.GetOwnerClassQN(mr.gc, container)
		return helper.IsSubClassOf(mr.gc, mr.fc, sourceClass, target.ParentQN)
	}

	return false
}
