package java

import (
	"fmt"
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// =============================================================================
// 原生 Java 语法糖处理 (Native Java Syntactic Sugar)
// =============================================================================

func (c *Collector) desugarDefaultConstructor(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	body := node.ChildByFieldName("body")
	if body == nil {
		return
	}
	for i := 0; i < int(body.NamedChildCount()); i++ {
		if body.NamedChild(uint(i)).Kind() == "constructor_declaration" {
			return
		}
	}

	consName := elem.Name
	consQN := c.resolver.BuildQualifiedName(elem.QualifiedName, consName+"()")
	fCtx.AddDefinition(&model.CodeElement{
		Kind:          model.Method,
		Name:          consName,
		QualifiedName: consQN,
		Path:          fCtx.FilePath,
		Location:      elem.Location,
		Signature:     fmt.Sprintf("public %s()", consName),
		Extra: &model.Extra{
			Modifiers:   make([]string, 0),
			Annotations: make([]string, 0),
			Mores:       map[string]interface{}{MethodIsConstructor: true, MethodIsImplicit: true},
		},
		IsFormSugar: true,
	}, elem.QualifiedName, node)
}

func (c *Collector) desugarEnumMethods(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	vQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "values()")
	fCtx.AddDefinition(&model.CodeElement{
		Kind: model.Method, Name: "values", QualifiedName: vQN, Path: fCtx.FilePath, Location: elem.Location, IsFormSugar: true,
		Signature: fmt.Sprintf("public static %s[] values()", elem.Name),
		Extra: &model.Extra{
			Modifiers:   make([]string, 0),
			Annotations: make([]string, 0),
			Mores:       map[string]interface{}{MethodIsImplicit: true},
		},
	}, elem.QualifiedName, node)

	voQN := c.resolver.BuildQualifiedName(elem.QualifiedName, "valueOf(String)")
	fCtx.AddDefinition(&model.CodeElement{
		Kind: model.Method, Name: "valueOf", QualifiedName: voQN, Path: fCtx.FilePath, Location: elem.Location, IsFormSugar: true,
		Signature: fmt.Sprintf("public static %s valueOf(String name)", elem.Name),
		Extra: &model.Extra{
			Modifiers:   make([]string, 0),
			Annotations: make([]string, 0),
			Mores:       map[string]interface{}{MethodIsImplicit: true},
		},
	}, elem.QualifiedName, node)
}

func (c *Collector) desugarRecordMembers(elem *model.CodeElement, node *sitter.Node, fCtx *core.FileContext) {
	paramList := c.findNamedChildOfType(node, "formal_parameters")
	if paramList == nil {
		return
	}

	type component struct{ name, vType string }
	var comps []component
	for i := 0; i < int(paramList.NamedChildCount()); i++ {
		child := paramList.NamedChild(uint(i))
		if child.Kind() == "formal_parameter" {
			comps = append(comps, component{
				name:  c.getNodeContent(child.ChildByFieldName("name"), *fCtx.SourceBytes),
				vType: c.getNodeContent(child.ChildByFieldName("type"), *fCtx.SourceBytes),
			})
		}
	}

	for _, comp := range comps {
		fieldQN := c.resolver.BuildQualifiedName(elem.QualifiedName, comp.name)
		if defs, _ := fCtx.FindByShortName(comp.name); len(defs) > 0 {
			for _, d := range defs {
				if d.Element.QualifiedName == fieldQN {
					d.Element.Kind = model.Field
					d.Element.Extra.Mores[FieldIsRecordComponent] = true
					d.Element.Extra.Mores[FieldIsFinal] = true
				}
			}
		}
		mIdentity := comp.name + "()"
		mQN := c.resolver.BuildQualifiedName(elem.QualifiedName, mIdentity)
		if len(c.findDefinitionsByQN(fCtx, mQN)) == 0 {
			fCtx.AddDefinition(&model.CodeElement{
				Kind: model.Method, Name: comp.name, QualifiedName: mQN, Path: fCtx.FilePath, Location: elem.Location, IsFormSugar: true,
				Signature: fmt.Sprintf("public %s %s()", comp.vType, comp.name),
				Extra: &model.Extra{
					Modifiers:   make([]string, 0),
					Annotations: make([]string, 0),
					Mores:       map[string]interface{}{MethodIsImplicit: true},
				},
			}, elem.QualifiedName, node)
		}
	}

	var pTypes []string
	for _, comp := range comps {
		pTypes = append(pTypes, strings.TrimSpace(strings.Split(comp.vType, "<")[0]))
	}
	cIdentity := fmt.Sprintf("%s(%s)", elem.Name, strings.Join(pTypes, ","))
	cQN := c.resolver.BuildQualifiedName(elem.QualifiedName, cIdentity)
	if len(c.findDefinitionsByQN(fCtx, cQN)) == 0 {
		fCtx.AddDefinition(&model.CodeElement{
			Kind: model.Method, Name: elem.Name, QualifiedName: cQN, Path: fCtx.FilePath, Location: elem.Location, IsFormSugar: true,
			Signature: fmt.Sprintf("public %s(%s)", elem.Name, c.getNodeContent(paramList, *fCtx.SourceBytes)),
			Extra: &model.Extra{
				Modifiers:   make([]string, 0),
				Annotations: make([]string, 0),
				Mores:       map[string]interface{}{MethodIsConstructor: true, MethodIsImplicit: true},
			},
		}, elem.QualifiedName, node)
	}
}
