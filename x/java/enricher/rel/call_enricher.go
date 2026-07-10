package rel

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type CallEnricher struct {
	fCtx *core.FileContext
	gCtx *core.GlobalContext
}

func (e *CallEnricher) EnrichMetadata(rel *model.DependencyRelation) {
	node, _ := rel.Mores[constants.TmpNode].(*sitter.Node)
	ctxNode, _ := rel.Mores[constants.TmpCtxNode].(*sitter.Node)
	if node == nil || ctxNode == nil {
		return
	}

	// 基础元数据
	rel.Mores[constants.RelCallIsStatic] = false
	rel.Mores[constants.RelCallIsConstructor] = false
	rel.Mores[constants.RelCallIsChained] = false

	if node == nil {
		return
	}

	// 补全方法名括号，使其符合 collector 规范
	if rel.Target != nil && rel.Target.Kind == model.Method && !strings.HasSuffix(rel.Target.QualifiedName, ")") {
		rel.Target.QualifiedName += "()"
	}

	// EnclosingMethod 溯源 (Lambda/匿名类溯源到所属方法)
	if rel.Source != nil {
		qn := rel.Source.QualifiedName
		stopMarkers := []string{".lambda", ".anonymousClass", "$", ".block"}
		for _, marker := range stopMarkers {
			if idx := strings.Index(qn, marker); idx != -1 {
				rel.Mores[constants.RelCallEnclosingMethod] = qn[:idx]
				break
			}
		}
	}
}

// _normalizeReceiverText 标准化receiver文本（去除换行符和多余空格）
func (e *CallEnricher) _normalizeReceiverText(raw string) string {
	normalized := ""
	for _, step := range strings.Split(raw, ".") {
		normalized += "." + strings.TrimSpace(step)
	}

	return strings.TrimLeft(normalized, ".")
}
