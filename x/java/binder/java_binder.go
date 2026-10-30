// Package binder 实现 Java 语言 core.Binder 接口：
// 将元素上收集到的 raw 类型字符串（返回值/参数/异常/变量类型）
// 解析为 QualifiedName，写入 WithQN 后缀的 Mores 键，建立全局符号索引。
package binder

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/helper"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/symbol_resolver"
)

// Binder 实现 core.Binder。
type Binder struct {
	resolver core.SymbolResolver
}

// New 创建 Java Binder。
func New() *Binder {
	return &Binder{
		resolver: symbol_resolver.New(),
	}
}

// BindSymbols 遍历全局上下文中的全部定义，完成 raw 类型到 QN 的绑定。
// 注意：GlobalContext 内部自加锁，此处通过快照遍历，不持有任何锁。
func (b *Binder) BindSymbols(gc *core.GlobalContext) error {
	for _, fCtx := range gc.FileContexts() {
		for _, entry := range fCtx.Definitions {
			elem := entry.Element
			if elem.Extra == nil {
				continue
			}

			// 1. 返回值消解（带 owner 作用域，优先解析 owner 的兄弟嵌套类）
			if raw, ok := elem.Extra.Mores[constants.MethodReturnType].(string); ok {
				elem.Extra.Mores[constants.MethodReturnTypeWithQN] = b.resolveToQNScoped(gc, fCtx, elem.QualifiedName, raw)
			}

			// 2. 参数列表消解 ["type name", "type name", ...]
			if params, ok := elem.Extra.Mores[constants.MethodParameters].([]string); ok {
				qnParams := make([]string, len(params))
				copy(qnParams, params)
				for i := 0; i < len(params); i++ {
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
	return nil
}

// resolveToQN 将 raw 类型字符串解析为 QN；基本类型原样返回。
func (b *Binder) resolveToQN(gc *core.GlobalContext, fCtx *core.FileContext, rawType string) string {
	return b.resolveToQNScoped(gc, fCtx, "", rawType)
}

// resolveToQNScoped 将 raw 类型字符串解析为 QN，ownerQN 用于短名作用域限定。
// 短名（如 Builder）在文件内 FindByShortName 可能命中多个同名嵌套类（外层 Builder 与
// 嵌套 Config.Builder），file-order 首个不一定是当前 owner 作用域下的嵌套类。
// 算法：沿词法嵌套链逐级上溯（当前类 -> 外层类 -> 更外层），尝试 scope+shortName；
// 例：Config.Builder 内方法返回 Builder -> Config.Builder；Config.builder() 返回 Builder
// -> Config.Builder；BuilderComprehensive.builder() 返回 Builder -> BuilderComprehensive.Builder。
func (b *Binder) resolveToQNScoped(gc *core.GlobalContext, fCtx *core.FileContext, ownerQN, rawType string) string {
	if b.resolver.IsPrimitive(rawType) {
		return rawType
	}
	rawType = strings.TrimSpace(rawType)
	if rawType == "" {
		return ""
	}

	// 短名 + owner 作用域：沿词法嵌套链上溯查找。
	if !strings.Contains(rawType, ".") && ownerQN != "" {
		owner := helper.GetOwnerClassQN(gc, &model.CodeElement{QualifiedName: ownerQN})
		for scope := owner; scope != ""; {
			scoped := scope + "." + rawType
			if entry, ok := gc.FindByQualifiedName(scoped); ok {
				return entry.Element.QualifiedName
			}
			parentEntry, ok := gc.FindByQualifiedName(scope)
			if !ok {
				break
			}
			scope = parentEntry.ParentQN
		}
	}

	ele := b.resolver.ResolveType(gc, fCtx, rawType, model.Class)
	if ele == nil {
		return rawType // 泛型变量/通配符等非类型：保留原始文本，QN 绑定失败不致命
	}
	return ele.QualifiedName
}
