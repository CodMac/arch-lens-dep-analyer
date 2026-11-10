package context

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/helper"
)

// GlobalContext 是 C/C++ 的全局符号上下文。
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

// RegisterFileContext 先注册命名空间层级，再登记文件内定义。
func (gc *GlobalContext) RegisterFileContext(fc core.FileContext) {
	gc.RegisterPackage(fc.PackageName())
	gc.BaseGlobalContext.RegisterFileContext(fc)
}

// RegisterPackage 将 :: 分隔的命名空间逐级注册为 NAMESPACE 元素。
func (gc *GlobalContext) RegisterPackage(packageName string) {
	if packageName == "" {
		return
	}
	parts := helper.SplitQN(packageName)
	var cur []string
	for _, p := range parts {
		if p == "" || p == "$anon" {
			continue
		}
		cur = append(cur, p)
		qn := strings.Join(cur, helper.QNSep)
		if _, ok := gc.FindByQualifiedName(qn); !ok {
			gc.AddDefinition(&core.DefinitionEntry{
				Element: &model.CodeElement{
					Kind: model.Namespace, Name: p, QualifiedName: qn, IsFormSource: true,
				},
			})
		}
	}
}
