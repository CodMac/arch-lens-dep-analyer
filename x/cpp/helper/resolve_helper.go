package helper

import (
	"github.com/CodMac/arch-lens-dep-analyer/model"
)

// IsScopeContainer 判断元素 kind 是否为作用域容器。
// 作用域容器拥有独立 QN 前缀，其内部定义继续以其 QN 为父级。
func IsScopeContainer(k model.ElementKind) bool {
	switch k {
	case model.Function, model.Method, model.FunctionDecl, model.MethodDecl,
		model.Struct, model.Class, model.Namespace, model.Enum,
		model.ScopeBlock, model.Lambda:
		return true
	}
	return false
}
