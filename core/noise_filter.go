package core

import (
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// NoiseFilter 负责按过滤等级移除噪音关系。
type NoiseFilter interface {
	// SetLevel 设置过滤等级。
	SetLevel(level FilterLevel)

	// IsNoise 判断给定关系是否为噪音。
	IsNoise(rel model.DependencyRelation) bool
}

// DefaultNoiseFilter 提供等级存储的默认实现，语言过滤逻辑可内嵌复用以免重复存储 Level。
type DefaultNoiseFilter struct {
	Level FilterLevel
}

// SetLevel 设置过滤等级。
func (d *DefaultNoiseFilter) SetLevel(level FilterLevel) { d.Level = level }
