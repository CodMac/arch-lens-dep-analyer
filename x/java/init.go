// Package java 提供 Java 语言的完整 SPI 实现聚合注册：
// 通过 core.Register 一次性注册 Parser/Collector/Binder/Extractor/Linker/
// NoiseFilter/SymbolResolver/SymbolBuilder 全套实现。
package java

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/binder"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/collector"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/extractor"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/linker"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/noise_filter"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/symbol_builder"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/symbol_resolver"
)

func init() {
	resolver := symbol_resolver.New()

	core.Register(&core.LanguageImpl{
		Language:       core.LangJava,
		Parser:         &JavaParserFactory{},
		Collector:      collector.New(resolver),
		Binder:         binder.New(),
		Extractor:      extractor.New(),
		Linker:         linker.New(),
		NoiseFilter:    noise_filter.New(core.LevelBalanced),
		SymbolResolver: resolver,
		SymbolBuilder:  symbol_builder.New(),
	})
}
