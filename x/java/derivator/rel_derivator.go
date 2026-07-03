package derivator

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
)

type RelDerivator struct {
	fCtx *core.FileContext
	gCtx *core.GlobalContext

	resolver core.SymbolResolver
	builder  core.SymbolBuilder
}

func NewRelDerivator(fCtx *core.FileContext, gCtx *core.GlobalContext) *RelDerivator {
	return &RelDerivator{
		fCtx: fCtx,
		gCtx: gCtx,

		resolver: gCtx.Resolver,
		builder:  gCtx.Builder,
	}
}

// DeriveCaptureRelations 基于一阶基础关系(ASSIGN/USE)的高级特征，生成派生的 Capture 二阶关系
func (p *RelDerivator) DeriveCaptureRelations(deps []*model.DependencyRelation) []*model.DependencyRelation {
	var captures []*model.DependencyRelation
	seen := make(map[string]bool)

	for _, rel := range deps {
		if rel.Source == nil || rel.Target == nil {
			continue
		}

		isCapture := false
		if rel.Type == model.Use {
			if val, ok := rel.Mores[constants.RelUseIsCapture].(bool); ok && val {
				isCapture = true
			}
		}
		if rel.Type == model.Assign {
			if val, ok := rel.Mores[constants.RelAssignIsCapture].(bool); ok && val {
				isCapture = true
			}
		}

		if isCapture {
			key := rel.Source.QualifiedName + "->" + rel.Target.QualifiedName
			if !seen[key] {
				seen[key] = true
				captures = append(captures, &model.DependencyRelation{
					Source:   rel.Source,
					Target:   rel.Target,
					Type:     model.Capture,
					Location: rel.Location,
					Mores:    make(map[string]interface{}),
				})
			}
		}
	}
	return captures
}

// SupplementTypeArgs 对定义块进行高级二次扫描，提取潜藏的泛型泛参依赖关系
func (p *RelDerivator) SupplementTypeArgs(definitions []*core.DefinitionEntry) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	for _, entry := range definitions {
		elem := entry.Element
		if elem.Extra == nil {
			continue
		}

		// 搜集该元素上所有可能带有泛型的原始声明文本
		var rawTypes []string
		keys := []string{constants.FieldRawType, constants.VariableRawType, constants.MethodReturnType}
		for _, k := range keys {
			if v, ok := elem.Extra.Mores[k].(string); ok {
				rawTypes = append(rawTypes, v)
			}
		}
		if pts, ok := elem.Extra.Mores[constants.MethodParameters].([]string); ok {
			for _, paramStr := range pts {
				// 简易抓取参数类型部分
				parts := strings.Fields(paramStr)
				if len(parts) >= 2 {
					rawTypes = append(rawTypes, parts[len(parts)-2])
				} else {
					rawTypes = append(rawTypes, paramStr)
				}
			}
		}

		// 执行递归搜集
		for _, rt := range rawTypes {
			rels = append(rels, p.collectAllTypeArgs(rt, elem)...)
		}
	}
	return rels
}

func (p *RelDerivator) collectAllTypeArgs(rt string, source *model.CodeElement) []*model.DependencyRelation {
	var rels []*model.DependencyRelation
	if !strings.Contains(rt, "<") {
		return nil
	}

	args := p.parseTypeArgs(rt)
	for i, arg := range args {
		target := p.resolver.ResolveType(p.gCtx, p.fCtx, helper.Clean(arg), model.Class)
		rels = append(rels, &model.DependencyRelation{
			Type:   model.TypeArg,
			Source: source,
			Target: target,
			Mores:  map[string]interface{}{constants.RelTypeArgIndex: i, constants.RelRawText: arg, constants.RelNodeAstKind: "type_arguments"},
		})

		if strings.Contains(arg, "<") {
			rels = append(rels, p.collectAllTypeArgs(arg, source)...)
		}
	}
	return rels
}

func (p *RelDerivator) parseTypeArgs(rawType string) []string {
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
