// Package core 定义语言无关的核心 SPI 契约。
package core

import (
	"fmt"
)

// Language 表示一种受支持的编程语言。
type Language string

const (
	LangJava Language = "java"
	LangGo   Language = "go"
)

// FilterLevel 表示噪音过滤等级。
type FilterLevel int

const (
	// LevelRaw 直通，不过滤。
	LevelRaw FilterLevel = iota
	// LevelBalanced 平衡：过滤外部符号的部分关系。
	LevelBalanced
	// LevelPure 纯净：仅保留源码符号相关关系。
	LevelPure
)

// LanguageImpl 聚合一个语言的全部 SPI 实现。
// 各语言在 init() 中构造并调用 Register 完成注册。
type LanguageImpl struct {
	Language       Language
	Parser         ParserFactory
	Collector      Collector
	Binder         Binder
	Extractor      Extractor
	Linker         Linker
	NoiseFilter    NoiseFilter
	SymbolResolver SymbolResolver
	SymbolBuilder  SymbolBuilder
}

// registry 全局语言注册表。
var registry = struct {
	impls map[Language]*LanguageImpl
}{
	impls: make(map[Language]*LanguageImpl),
}

// Register 注册一个语言实现。重复注册同一语言会覆盖。
func Register(impl *LanguageImpl) {
	if impl == nil {
		return
	}
	registry.impls[impl.Language] = impl
}

// Get 获取指定语言的实现，未注册则返回错误。
func Get(lang Language) (*LanguageImpl, error) {
	impl, ok := registry.impls[lang]
	if !ok {
		return nil, fmt.Errorf("no language implementation registered for %q", lang)
	}
	return impl, nil
}

// GetOrPanic 获取语言实现，未注册时 panic（用于已知必有的场景）。
func GetOrPanic(lang Language) *LanguageImpl {
	impl, err := Get(lang)
	if err != nil {
		panic(err)
	}
	return impl
}
