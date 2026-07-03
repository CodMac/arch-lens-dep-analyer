package resolver

import (
	"slices"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

type MemberResolver struct {
	gCtx *core.GlobalContext
	fCtx *core.FileContext
}

func NewMemberResolver(gCtx *core.GlobalContext, fCtx *core.FileContext) *MemberResolver {
	return &MemberResolver{
		gCtx: gCtx,
		fCtx: fCtx,
	}
}

func (mr *MemberResolver) ResolveField(currType *model.CodeElement, name string, isStatic bool, fromCtx *model.CodeElement) *model.CodeElement {
	if currType == nil || name == "" {
		return nil
	}

	targetQN := currType.QualifiedName + "." + name
	if entry, ok := mr.gCtx.FindByQualifiedName(targetQN); ok && entry.Element.Kind == model.Field {
		if mr.checkVisibility(fromCtx, entry) {
			if !isStatic || slices.Contains(entry.Element.Extra.Modifiers, "static") {
				return entry.Element
			}
		}
	}

	return mr.searchFieldInInheritance(currType, name, isStatic, fromCtx)
}

func (mr *MemberResolver) ResolveMethod(currType *model.CodeElement, name string, argTypes []string, isStatic bool, fromCtx *model.CodeElement) *model.CodeElement {
	if currType == nil || name == "" {
		return nil
	}

	var candidates []*core.DefinitionEntry
	mr.collectMethodCandidates(currType, name, isStatic, fromCtx, &candidates, make(map[string]bool))

	if len(candidates) == 0 {
		return nil
	}

	return mr.pickBestOverload(candidates, argTypes)
}

func (mr *MemberResolver) searchFieldInInheritance(elem *model.CodeElement, name string, isStatic bool, fromCtx *model.CodeElement) *model.CodeElement {
	if elem.Extra == nil {
		return nil
	}

	var supers []string
	if sc, ok := elem.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
		supers = append(supers, sc)
	}
	if itfs, ok := elem.Extra.Mores[constants.ClassImplementedInterfaces].([]string); ok {
		supers = append(supers, itfs...)
	}

	for _, superName := range supers {
		parents := helper.PreciseResolve(mr.gCtx, mr.fCtx, helper.Clean(superName))
		for _, p := range parents {
			if found := mr.ResolveField(p.Element, name, isStatic, fromCtx); found != nil {
				return found
			}
		}
	}
	return nil
}

func (mr *MemberResolver) collectMethodCandidates(elem *model.CodeElement, name string, isStatic bool, fromCtx *model.CodeElement, candidates *[]*core.DefinitionEntry, visited map[string]bool) {
	if elem == nil || visited[elem.QualifiedName] {
		return
	}
	visited[elem.QualifiedName] = true

	targetPrefix := elem.QualifiedName + "." + name
	if entries, ok := mr.gCtx.FindMethodByNoParamsQN(targetPrefix); ok {
		for _, e := range entries {
			if e.Element.Kind != model.Method {
				continue
			}
			if isStatic && !slices.Contains(e.Element.Extra.Modifiers, "static") {
				continue
			}
			if mr.checkVisibility(fromCtx, e) {
				*candidates = append(*candidates, e)
			}
		}
	}

	if elem.Extra == nil {
		return
	}

	var supers []string
	if sc, ok := elem.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
		supers = append(supers, sc)
	}
	if itfs, ok := elem.Extra.Mores[constants.ClassImplementedInterfaces].([]string); ok {
		supers = append(supers, itfs...)
	}

	for _, superName := range supers {
		parents := helper.PreciseResolve(mr.gCtx, mr.fCtx, helper.Clean(superName))
		for _, p := range parents {
			mr.collectMethodCandidates(p.Element, name, isStatic, fromCtx, candidates, visited)
		}
	}
}

func (mr *MemberResolver) pickBestOverload(candidates []*core.DefinitionEntry, argTypes []string) *model.CodeElement {
	var bestMatch *model.CodeElement
	maxScore := -1

	for _, candidate := range candidates {
		score := mr.calculateOverloadScore(candidate, argTypes)
		if score > maxScore {
			maxScore = score
			bestMatch = candidate.Element
		}
	}

	if bestMatch != nil && maxScore > 0 {
		return bestMatch
	}
	return candidates[0].Element
}

func (mr *MemberResolver) calculateOverloadScore(entry *core.DefinitionEntry, argTypes []string) int {
	params, ok := entry.Element.Extra.Mores[constants.MethodParametersWithQN].([]string)
	if !ok {
		return 0
	}

	definedCount := len(params)
	inferredCount := len(argTypes)
	score := 0

	isVarargs := false
	if definedCount > 0 && strings.HasSuffix(params[definedCount-1], "...") {
		isVarargs = true
	}

	if definedCount == inferredCount {
		score += 100
	} else if isVarargs && inferredCount >= definedCount-1 {
		score += 50
	} else {
		return -1
	}

	for i := 0; i < inferredCount; i++ {
		var definedType string
		if isVarargs && i >= definedCount-1 {
			definedType = helper.Clean(strings.TrimSuffix(params[definedCount-1], "..."))
		} else {
			definedType = helper.Clean(params[i])
		}

		inferredType := argTypes[i]
		if inferredType == "unknown" {
			score += 10
			continue
		}
		if inferredType == "null" {
			if !mr.isPrimitiveType(definedType) {
				score += 20
			}
			continue
		}

		if definedType == inferredType || strings.HasSuffix(definedType, "."+inferredType) {
			score += 50
			continue
		}

		if mr.isSubclassOrInterfaceOf(inferredType, definedType) {
			score += 40
			continue
		}

		if mr.isCompatiblePrimitiveOrBox(inferredType, definedType) {
			score += 30
			continue
		}

		return -1
	}

	return score
}

func (mr *MemberResolver) isSubclassOrInterfaceOf(subQN, superQN string) bool {
	if subQN == "" || superQN == "" {
		return false
	}
	if subQN == superQN || strings.HasSuffix(subQN, "."+superQN) {
		return true
	}

	entry, ok := mr.gCtx.FindByQualifiedName(subQN)
	if !ok || entry.Element.Extra == nil {
		return false
	}

	var supers []string
	if sc, ok := entry.Element.Extra.Mores[constants.ClassSuperClass].(string); ok && sc != "" {
		supers = append(supers, sc)
	}
	if itfs, ok := entry.Element.Extra.Mores[constants.ClassImplementedInterfaces].([]string); ok {
		supers = append(supers, itfs...)
	}

	for _, parentName := range supers {
		parents := helper.PreciseResolve(mr.gCtx, mr.fCtx, helper.Clean(parentName))
		for _, p := range parents {
			if mr.isSubclassOrInterfaceOf(p.Element.QualifiedName, superQN) {
				return true
			}
		}
	}
	return false
}

func (mr *MemberResolver) isCompatiblePrimitiveOrBox(inferred, defined string) bool {
	primitives := map[string][]string{
		"int":    {"long", "float", "double", "java.lang.Integer", "java.lang.Long"},
		"long":   {"float", "double", "java.lang.Long"},
		"float":  {"double", "java.lang.Float"},
		"char":   {"int", "long", "java.lang.Character"},
		"byte":   {"short", "int", "long", "java.lang.Byte"},
		"short":  {"int", "long", "java.lang.Short"},
		"double": {"java.lang.Double"},
	}
	targets, ok := primitives[inferred]
	if !ok {
		return false
	}
	return slices.Contains(targets, defined) || slices.Contains(targets, "java.lang."+defined)
}

func (mr *MemberResolver) isPrimitiveType(t string) bool {
	pt := []string{"int", "long", "double", "float", "boolean", "char", "byte", "short"}
	return slices.Contains(pt, t)
}

func (mr *MemberResolver) checkVisibility(container *model.CodeElement, target *core.DefinitionEntry) bool {
	if target.Element.Kind == model.Variable {
		return true
	}
	if container == nil || target.Element == nil {
		return false
	}
	cOutermost := helper.GetOutermostClassQN(container.QualifiedName)
	tOutermost := helper.GetOutermostClassQN(target.Element.QualifiedName)
	if cOutermost != "" && cOutermost == tOutermost {
		return true
	}
	if target.Element.Extra == nil || target.Element.Extra.Modifiers == nil {
		return false
	}
	mods := target.Element.Extra.Modifiers
	if slices.Contains(mods, "public") {
		return true
	}
	if helper.GetRealPackage(mr.gCtx, target.Element) == mr.fCtx.PackageName {
		return true
	}
	if slices.Contains(mods, "protected") {
		srcClass := helper.GetOwnerClassQN(mr.gCtx, container)
		return helper.IsSubClassOf(mr.gCtx, mr.fCtx, srcClass, target.ParentQN)
	}
	return false
}
