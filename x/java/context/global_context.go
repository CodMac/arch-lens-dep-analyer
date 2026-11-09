package context

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// GlobalContext 是 Java 的全局符号上下文。
type GlobalContext struct {
	*core.BaseGlobalContext
}

// NewGlobalContext 创建 Java 全局上下文。
func NewGlobalContext(resolver core.SymbolResolver, builder core.SymbolBuilder) core.GlobalContext {
	return &GlobalContext{
		BaseGlobalContext: core.NewBaseGlobalContext(resolver, builder, core.GlobalContextRules{
			// Java 只有方法参与重载消解。
			OverloadKinds: map[model.ElementKind]struct{}{
				model.Method: {},
			},
		}),
	}
}

// RegisterFileContext 先注册包层级，再登记文件内定义。
func (gc *GlobalContext) RegisterFileContext(fc core.FileContext) {
	gc.RegisterPackage(fc.PackageName())
	gc.BaseGlobalContext.RegisterFileContext(fc)
}

// RegisterPackage 将点号包名拆分为逐级 PACKAGE 元素。
func (gc *GlobalContext) RegisterPackage(packageName string) {
	parts := strings.Split(packageName, ".")
	var current []string
	for _, part := range parts {
		if part == "" {
			continue
		}
		current = append(current, part)
		pkgQN := strings.Join(current, ".")
		if _, ok := gc.FindByQualifiedName(pkgQN); !ok {
			gc.AddDefinition(&core.DefinitionEntry{
				Element: &model.CodeElement{Kind: model.Package, Name: part, QualifiedName: pkgQN, IsFormSource: true},
			})
		}
	}
}
