// Package noise_filter 实现 Java 语言 core.NoiseFilter 接口：
// 依据目标符号来源（源码/语法糖/外部）与关系类型，按 Balanced/Pure 等级过滤噪音关系。
package noise_filter

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// NoiseFilter 实现 core.NoiseFilter，内嵌默认等级存储。
type NoiseFilter struct {
	core.DefaultNoiseFilter
}

// New 创建 Java NoiseFilter（默认 Balanced 等级）。
func New(level core.FilterLevel) *NoiseFilter {
	return &NoiseFilter{
		DefaultNoiseFilter: core.DefaultNoiseFilter{Level: level},
	}
}

// IsNoise 判定关系是否为噪音。
func (f *NoiseFilter) IsNoise(rel model.DependencyRelation) bool {
	if rel.Target == nil || f.Level == core.LevelRaw {
		return false
	}

	// 规则 1 & 2: 源码、语法糖、包含关系 始终保留
	if rel.Target.IsFormSource || rel.Target.IsFormSugar || rel.Type == model.Contain {
		return false
	}

	// 进入 LevelBalanced 的判定
	if f.Level == core.LevelBalanced {
		// 目标是外部符号
		if rel.Target.IsFormExternal {
			switch rel.Type {
			// 规则 3: 外部继承/实现 -> 噪音
			case model.Extend, model.Implement:
				return true
			// 规则 4: 外部类型引用 -> 噪音
			case model.Annotation, model.Parameter, model.Return, model.TypeArg:
				return true
			// 规则 5: 外部行为与执行流 -> 噪音
			case model.Call, model.Create, model.Cast, model.Use, model.Assign:
				return true
			}
		}
	}

	// LevelPure 逻辑
	if f.Level == core.LevelPure {
		return rel.Target.IsFormExternal
	}

	return false
}
