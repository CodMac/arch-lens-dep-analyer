// Package cpp 提供 C/C++ 语言的完整 SPI 实现聚合注册：
// 通过 core.Register 注册 Parser/Collector/Binder/Extractor/Linker/
// NoiseFilter/SymbolResolver/SymbolBuilder 全套实现。
//
// C 与 C++ 共用同一套处理逻辑（tree-sitter-cpp 是 C 的严格超集），
// 仅 Parser 层按文件扩展名选择 grammar。
package cpp

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/binder"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/collector"
	cppctx "github.com/CodMac/arch-lens-dep-analyer/x/cpp/context"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/extractor"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/linker"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/noise_filter"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/symbol_builder"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/symbol_resolver"
)

func init() {
	resolver := symbol_resolver.New()
	builder := symbol_builder.New()

	impl := func(lang core.Language) *core.LanguageImpl {
		return &core.LanguageImpl{
			Language:       lang,
			Parser:         &CppParserFactory{},
			Collector:      collector.New(resolver),
			Binder:         binder.New(resolver),
			Extractor:      extractor.New(resolver),
			Linker:         linker.New(),
			NoiseFilter:    noise_filter.New(core.LevelBalanced),
			SymbolResolver: resolver,
			SymbolBuilder:  builder,

			NewGlobalContext: cppctx.NewGlobalContext,
		}
	}

	core.Register(impl(core.LangC))
	core.Register(impl(core.LangCPP))
}
