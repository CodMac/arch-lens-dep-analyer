package derived

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

// TypeArgExtractor 派生 TypeArg（泛型实参/边界）关系。
// 输入源为元素级 Mores 的原始类型文本（类/接口/方法/字段/变量/参数），
// 既包含类型参数边界（T extends X），也包含泛型实参（List<X> 的 X）。
type TypeArgExtractor struct{}

// Extract 从当前文件全部定义中派生 TypeArg 关系。
func (e *TypeArgExtractor) Extract(ctx *Context) []*model.DependencyRelation {
	var rels []*model.DependencyRelation

	for _, entry := range ctx.FC.Definitions {
		elem := entry.Element
		if elem.Extra == nil {
			continue
		}

		// 1. 现有：字段/变量/方法返回/参数的泛型实参（如 List<String> 的 String）。
		for _, rt := range e.getRawTypesForTypeArgs(elem) {
			rels = append(rels, e.collectAllTypeArgs(rt, elem, ctx)...)
		}

		// 2. 类/接口类型参数边界（如 <T extends AbstractBaseEntity<?>> 的 AbstractBaseEntity）。
		if tp, ok := elem.Extra.Mores[constants.ClassTypeParameters].(string); ok && tp != "" {
			rels = append(rels, e.collectTypeParameterBounds(tp, elem, ctx)...)
		}

		// 3. 父类/接口泛型实参（如 extends AbstractBaseEntity<String> 的 AbstractBaseEntity）。
		for _, sc := range e.getHeritageTypes(elem) {
			if strings.Contains(sc, "<") {
				rels = append(rels, e.collectAllTypeArgs(sc, elem, ctx)...)
			}
		}

		// 4. 方法泛型参数边界（如 <E extends Number> 的 Number）。
		if tp, ok := elem.Extra.Mores[constants.MethodTypeParameters].(string); ok && tp != "" {
			rels = append(rels, e.collectTypeParameterBounds(tp, elem, ctx)...)
		}
	}

	return rels
}

// getRawTypesForTypeArgs 收集用于泛型解析的原始类型文本（字段/变量/方法返回/参数）。
func (e *TypeArgExtractor) getRawTypesForTypeArgs(elem *model.CodeElement) (res []string) {
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

// extractTypeFromParam 从参数原始文本中提取类型部分。
func (e *TypeArgExtractor) extractTypeFromParam(p string) string {
	parts := strings.Fields(p)
	if len(parts) >= 2 {
		return parts[len(parts)-2]
	}
	return p
}

// parseTypeArgs 解析泛型实参（支持嵌套泛型）。
func (e *TypeArgExtractor) parseTypeArgs(rawType string) []string {
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

// collectAllTypeArgs 递归收集泛型实参对应的 TypeArg 关系。
func (e *TypeArgExtractor) collectAllTypeArgs(rt string, source *model.CodeElement, ctx *Context) []*model.DependencyRelation {
	var rels []*model.DependencyRelation

	if !strings.Contains(rt, "<") {
		return nil
	}

	args := e.parseTypeArgs(rt)
	for i, arg := range args {
		target := ctx.Resolver.ResolveType(ctx.GC, ctx.FC, helper.Clean(arg), model.Class)
		// 泛型变量（T/E/S）、通配符（?）解析为 nil，不产关系（非真实类型依赖）。
		if target == nil {
			continue
		}
		rels = append(rels, &model.DependencyRelation{
			Type: model.TypeArg, Source: source, Target: target,
			Mores: map[string]interface{}{constants.RelTypeArgIndex: i, constants.RelRawText: arg, constants.RelAstKind: "type_arguments"},
		})

		if strings.Contains(arg, "<") {
			rels = append(rels, e.collectAllTypeArgs(arg, source, ctx)...)
		}
	}
	return rels
}

// collectTypeParameterBounds 从类型参数声明（如 <T extends AbstractBaseEntity<?>>）提取上界产出 TypeArg。
func (e *TypeArgExtractor) collectTypeParameterBounds(typeParams string, source *model.CodeElement, ctx *Context) []*model.DependencyRelation {
	var rels []*model.DependencyRelation

	for _, p := range e.parseTypeArgs(typeParams) { // 分割 <...> 内逗号
		idx := strings.Index(p, "extends")
		if idx < 0 {
			continue // 无界参数（T/ID）跳过
		}
		for _, bound := range strings.Split(p[idx+len("extends"):], "&") { // 多边界
			bound = strings.TrimSpace(bound)
			if bound == "" || bound == "?" {
				continue
			}
			target := ctx.Resolver.ResolveType(ctx.GC, ctx.FC, helper.Clean(bound), model.Class)
			if target == nil {
				continue
			}
			rels = append(rels, &model.DependencyRelation{
				Type: model.TypeArg, Source: source, Target: target,
				Mores: map[string]interface{}{
					constants.RelRawText:      bound,
					constants.RelAstKind:      "type_parameters",
					constants.RelTypeArgIndex: 0,
				},
			})
			if strings.Contains(bound, "<") {
				rels = append(rels, e.collectAllTypeArgs(bound, source, ctx)...) // 边界自身泛型
			}
		}
	}
	return rels
}

// getHeritageTypes 收集类/接口继承的原始类型文本（父类 1 个 + 实现接口 N 个）。
func (e *TypeArgExtractor) getHeritageTypes(elem *model.CodeElement) (res []string) {
	if sc, ok := elem.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
		res = append(res, sc)
	}
	if impls, ok := elem.Extra.Mores[constants.ClassImplementedInterfaces].([]string); ok {
		res = append(res, impls...)
	}
	if impls, ok := elem.Extra.Mores[constants.InterfaceImplementedInterfaces].([]string); ok {
		res = append(res, impls...)
	}
	return
}
