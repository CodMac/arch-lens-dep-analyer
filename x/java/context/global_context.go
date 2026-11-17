package context

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
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
	if pfc, ok := fc.(core.PackageAwareFileContext); ok {
		if name := pfc.PackageName(); name != "" {
			gc.registerPackage(name)
		} else {
			gc.registerDefaultPackage()
		}
	}
	gc.BaseGlobalContext.RegisterFileContext(fc)
}

// registerPackage 将点号包名拆分为逐级 PACKAGE 元素（Java 专属语义）。
func (gc *GlobalContext) registerPackage(packageName string) {
	parts := strings.Split(packageName, ".")
	var current []string
	for _, part := range parts {
		if part == "" {
			continue
		}
		current = append(current, part)
		pkgQN := strings.Join(current, ".")
		gc.addPackageElem(part, pkgQN)
	}
}

// registerDefaultPackage 为默认包（无 package 声明）注册哨兵 PACKAGE 元素。
func (gc *GlobalContext) registerDefaultPackage() {
	gc.addPackageElem(constants.DefaultPackageQN, constants.DefaultPackageQN)
}

// addPackageElem 注册一个 PACKAGE 元素（幂等）。
func (gc *GlobalContext) addPackageElem(name, pkgQN string) {
	if _, ok := gc.FindByQualifiedName(pkgQN); ok {
		return
	}
	gc.AddDefinition(&core.DefinitionEntry{
		Element: &model.CodeElement{
			Kind:          model.Package,
			Name:          name,
			QualifiedName: pkgQN,
			IsFormSource:  true,
		},
	})
}
