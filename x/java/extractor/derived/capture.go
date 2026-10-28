package derived

import (
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
)

// CaptureExtractor 从 USE/ASSIGN 关系推导 CAPTURE 关系（跨作用域变量捕获）。
// 输入为已提取并经元数据增强的 Deps：is_capture 标记由 rel 增强器（use_enricher/assign_enricher）写入。
type CaptureExtractor struct{}

// Extract 从 ctx.Deps 的 is_capture 标记推导 CAPTURE。
func (e *CaptureExtractor) Extract(ctx *Context) []*model.DependencyRelation {
	var captures []*model.DependencyRelation
	seen := make(map[string]bool)

	for _, relObj := range ctx.Deps {
		if relObj.Source == nil || relObj.Target == nil {
			continue
		}

		isCapture := false

		if relObj.Type == model.Use {
			if val, ok := relObj.Mores[constants.RelUseIsCapture]; ok {
				if b, isBool := val.(bool); isBool && b {
					isCapture = true
				}
			}
		}

		if relObj.Type == model.Assign {
			if val, ok := relObj.Mores[constants.RelAssignIsCapture]; ok {
				if b, isBool := val.(bool); isBool && b {
					isCapture = true
				}
			}
		}

		if isCapture {
			key := relObj.Source.QualifiedName + "->" + relObj.Target.QualifiedName

			if !seen[key] {
				seen[key] = true
				captureRel := &model.DependencyRelation{
					Source:   relObj.Source,
					Target:   relObj.Target,
					Type:     model.Capture,
					Location: relObj.Location,
					Mores:    make(map[string]interface{}),
				}
				captures = append(captures, captureRel)
			}
		}
	}
	return captures
}
