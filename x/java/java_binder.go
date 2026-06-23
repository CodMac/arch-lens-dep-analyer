package java

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
)

type Binder struct {
	resolver *SymbolResolver
}

func NewJavaBinder() *Binder {
	return &Binder{
		resolver: NewJavaSymbolResolver(),
	}
}

func (b *Binder) BindSymbols(gc *core.GlobalContext) {
	gc.RLock()
	defer gc.RUnlock()

	for _, fCtx := range gc.FileContexts {
		for _, entry := range fCtx.Definitions {
			elem := entry.Element
			if elem.Extra == nil {
				continue
			}

			// 1. 返回值消解
			if raw, ok := elem.Extra.Mores[constants.MethodReturnType].(string); ok {
				elem.Extra.Mores[constants.MethodReturnTypeWithQN] = b.resolveToQN(gc, fCtx, raw)
			}

			// 2. 参数列表消解 ["type name", "type name", ...]
			if params, ok := elem.Extra.Mores[constants.MethodParameters].([]string); ok {
				qnParams := make([]string, len(params))
				copy(qnParams, params)
				for i := 0; i < len(params); i += 1 {
					fields := strings.Fields(params[i])
					if len(fields) != 2 {
						qnParams[i] = params[i]
						continue
					}

					// 可变参数类型处理
					if strings.HasSuffix(fields[0], "...") {
						fieldType := strings.ReplaceAll(fields[0], "...", "...")
						qnParams[i] = b.resolveToQN(gc, fCtx, fieldType) + "..."
						continue
					}

					// 默认类型处理
					qnParams[i] = b.resolveToQN(gc, fCtx, fields[0])
				}
				elem.Extra.Mores[constants.MethodParametersWithQN] = qnParams
			}

			// 3. 异常消解
			if throws, ok := elem.Extra.Mores[constants.MethodThrowsTypes].([]string); ok {
				qnThrows := make([]string, len(throws))
				for i, t := range throws {
					qnThrows[i] = b.resolveToQN(gc, fCtx, t)
				}
				elem.Extra.Mores[constants.MethodThrowsTypesWithQN] = qnThrows
			}

			// 4. 变量类型消解
			if raw, ok := elem.Extra.Mores[constants.VariableRawType].(string); ok {
				elem.Extra.Mores[constants.VariableTypeWithQN] = b.resolveToQN(gc, fCtx, raw)
			}
		}
	}
}

func (b *Binder) resolveToQN(gc *core.GlobalContext, fCtx *core.FileContext, rawType string) string {
	if b.resolver.IsPrimitive(rawType) {
		return rawType
	}

	ele := b.resolver.ResolveType(gc, fCtx, rawType, model.Class)
	return ele.QualifiedName
}
