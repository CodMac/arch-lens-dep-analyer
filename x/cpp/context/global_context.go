package context

import (
	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// GlobalContext 是 C/C++ 的全局符号上下文。
//
// C/C++ 没有「包声明」：NAMESPACE 元素由 Collector 在遍历 namespace_definition 时创建，
// 故 RegisterFileContext 直接委托 Base，无需额外注册步骤。
type GlobalContext struct {
	*core.BaseGlobalContext
}

// NewGlobalContext 创建 C/C++ 全局上下文。
func NewGlobalContext(resolver core.SymbolResolver, builder core.SymbolBuilder) core.GlobalContext {
	return &GlobalContext{
		BaseGlobalContext: core.NewBaseGlobalContext(resolver, builder, core.GlobalContextRules{
			// C/C++ 的自由函数与声明同样参与重载消解。
			OverloadKinds: map[model.ElementKind]struct{}{
				model.Method:       {},
				model.Function:     {},
				model.MethodDecl:   {},
				model.FunctionDecl: {},
			},
		}),
	}
}
