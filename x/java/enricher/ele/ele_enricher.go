package ele

import (
	"fmt"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

type Enricher interface {
	EnrichMetadata(elem *model.CodeElement, fCtx *core.FileContext, extra *model.Extra)
}

var enricherMap = make(map[model.ElementKind]Enricher)

// RegisterEnricher 注册一个类型对应的 Enricher
func RegisterEnricher(kind model.ElementKind, enricher Enricher) {
	enricherMap[kind] = enricher
}

// GetEnricher 根据类型获取对应的 Enricher 实例。
func GetEnricher(kind model.ElementKind) (Enricher, error) {
	enricher, ok := enricherMap[kind]
	if !ok {
		return nil, fmt.Errorf("no enricher registered for kind: %s", kind)
	}

	return enricher, nil
}
