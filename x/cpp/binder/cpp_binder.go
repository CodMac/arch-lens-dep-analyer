// Package binder 实现 C/C++ 语言 core.Binder 接口：
// 将元素上收集到的 raw 类型字符串（返回值/参数/变量/字段类型）
// 解析为 QualifiedName，写入 WithQN 后缀的 Mores 键。
//
// C/C++ 与 Java 的关键差异：
//   - QN 分隔符为 ::，短名作用域上溯沿 ns/类嵌套链
//   - typedef/using 别名链需先展开再解析
//   - 无 import 语句，跨文件类型靠全局 QN 与 include 表
package binder

import (
	"strings"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/cpp/helper"
)

// Binder 实现 core.Binder。
type Binder struct {
	resolver core.SymbolResolver
}

// New 创建 C/C++ Binder。
func New(resolver core.SymbolResolver) *Binder {
	return &Binder{resolver: resolver}
}

// BindSymbols 遍历全局上下文中的全部定义，完成 raw 类型到 QN 的绑定。
// GlobalContext 内部自加锁，此处通过快照遍历，不持有任何锁。
func (b *Binder) BindSymbols(gc core.GlobalContext) error {
	// 先构建全局 typedef/using 别名表（别名 QN -> 底层类型文本）。
	aliases := b.collectAliases(gc)

	for _, fCtx := range gc.FileContexts() {
		for _, entry := range fCtx.Definitions() {
			elem := entry.Element
			if elem.Extra == nil {
				continue
			}
			mores := elem.Extra.Mores

			// 1. 返回值
			if raw, ok := mores[constants.FunctionReturnType].(string); ok && raw != "" {
				mores[constants.FunctionReturnTypeWithQN] = b.resolveToQN(gc, fCtx, aliases, elem.QualifiedName, raw)
			}

			// 2. 参数列表 ["type name", ...]
			if params, ok := mores[constants.FunctionParameters].([]string); ok {
				qnParams := make([]string, len(params))
				copy(qnParams, params)
				for i, p := range params {
					if p == "..." {
						qnParams[i] = p
						continue
					}
					typePart, namePart := splitParam(p)
					if typePart == "" {
						continue
					}
					qn := b.resolveToQN(gc, fCtx, aliases, elem.QualifiedName, typePart)
					if namePart != "" {
						qnParams[i] = qn + " " + namePart
					} else {
						qnParams[i] = qn
					}
				}
				mores[constants.FunctionParametersWithQN] = qnParams
			}

			// 3. 变量类型
			if raw, ok := mores[constants.VariableRawType].(string); ok && raw != "" {
				mores[constants.VariableTypeWithQN] = b.resolveToQN(gc, fCtx, aliases, elem.QualifiedName, raw)
			}

			// 4. 字段类型
			if raw, ok := mores[constants.FieldRawType].(string); ok && raw != "" {
				mores[constants.FieldTypeWithQN] = b.resolveToQN(gc, fCtx, aliases, elem.QualifiedName, raw)
			}

			// 5. typedef/using 底层类型
			if raw, ok := mores[constants.TypeUnderlying].(string); ok && raw != "" {
				mores[constants.TypeUnderlying+"_with_qn"] = b.resolveToQN(gc, fCtx, aliases, elem.QualifiedName, raw)
			}
		}
	}
	return nil
}

// collectAliases 收集全局 typedef/using 别名表：别名 QN -> 底层类型文本。
func (b *Binder) collectAliases(gc core.GlobalContext) map[string]string {
	aliases := make(map[string]string)
	for _, def := range gc.Definitions() {
		elem := def.Element
		if elem == nil || elem.Kind != model.Type || elem.Extra == nil {
			continue
		}
		if raw, ok := elem.Extra.Mores[constants.TypeUnderlying].(string); ok && raw != "" {
			aliases[elem.QualifiedName] = raw
		}
	}
	return aliases
}

// resolveToQN 将 raw 类型字符串解析为 QN；基本类型原样返回。
// ownerQN 用于短名的词法作用域上溯。
func (b *Binder) resolveToQN(gc core.GlobalContext, fCtx core.FileContext, aliases map[string]string, ownerQN, rawType string) string {
	rawType = helper.NormalizeTypeText(rawType)
	if rawType == "" {
		return ""
	}

	base := helper.BaseTypeName(rawType)
	if base == "" {
		return rawType
	}
	if b.resolver.IsPrimitive(base) {
		return base
	}
	// 模板类型变量（T/E/S...）不是真实类型依赖。
	if helper.IsTypeVariable(base) {
		return base
	}

	// typedef/using 别名链展开（最多 8 层，防环）。
	base = b.expandAlias(gc, fCtx, aliases, base, 8)
	if base == "" {
		return rawType
	}
	if b.resolver.IsPrimitive(base) {
		return base
	}

	// 短名 + owner 作用域：沿词法嵌套链上溯（ns::Class::member -> ns::Class -> ns）。
	if !strings.Contains(base, helper.QNSep) && ownerQN != "" {
		for scope := helper.PkgOf(ownerQN); scope != ""; {
			scoped := helper.BuildQN(scope, base)
			if entry, ok := gc.FindByQualifiedName(scoped); ok {
				return entry.Element.QualifiedName
			}
			next := helper.PkgOf(scope)
			if next == scope {
				break
			}
			scope = next
		}
	}

	ele := b.resolver.ResolveType(gc, fCtx, base, model.Class)
	if ele == nil {
		return base
	}
	return ele.QualifiedName
}

// expandAlias 展开 typedef/using 别名链。
func (b *Binder) expandAlias(gc core.GlobalContext, fCtx core.FileContext, aliases map[string]string, name string, depth int) string {
	if depth <= 0 {
		return name
	}
	// 别名可能以短名或限定名登记；先按限定名、再按文件内短名、再按全局短名查找。
	var aliasQN string
	if _, ok := aliases[name]; ok {
		aliasQN = name
	} else if entry, ok := fCtx.FindByQualifiedName(name); ok && entry.Element.Kind == model.Type {
		aliasQN = entry.Element.QualifiedName
	} else if entries, ok := fCtx.FindByShortName(name); ok {
		for _, e := range entries {
			if e.Element.Kind == model.Type {
				aliasQN = e.Element.QualifiedName
				break
			}
		}
	}
	if aliasQN == "" {
		// 全局短名扫描（跨文件 typedef）。
		for _, def := range gc.Definitions() {
			if def.Element.Kind == model.Type && def.Element.Name == name {
				aliasQN = def.Element.QualifiedName
				break
			}
		}
	}
	underlying, ok := aliases[aliasQN]
	if !ok || underlying == "" {
		return name
	}
	// 匿名 struct/union/class 承载实体（typedef struct {...} Point;）：
	// 别名本身即目标类型，底层文本不是可解析的类型名，不展开。
	if isAnonymousTypeText(underlying) {
		return aliasQN
	}
	next := helper.BaseTypeName(underlying)
	if next == "" || next == name {
		return name
	}
	return b.expandAlias(gc, fCtx, aliases, next, depth-1)
}

// splitParam 将 "int x" / "const char* s" / "std::vector<int> v" 拆为类型与名称。
// 名称恒为最后一个 token（若最后一个 token 是合法标识符且前面还有内容）。
func splitParam(p string) (typePart, namePart string) {
	p = helper.NormalizeTypeText(p)
	if p == "" {
		return "", ""
	}
	// 函数指针参数：int (*cb)(int) —— 名称在 (*name) 内。
	if i := strings.Index(p, "(*"); i >= 0 {
		rest := p[i+2:]
		if j := strings.Index(rest, ")"); j > 0 {
			return p[:i] + "(*)", rest[:j]
		}
	}
	idx := strings.LastIndexAny(p, " \t")
	if idx < 0 {
		return p, ""
	}
	candidate := strings.TrimSpace(p[idx+1:])
	if candidate == "" || !isIdentifier(candidate) {
		return p, ""
	}
	return strings.TrimSpace(p[:idx]), candidate
}

// isAnonymousTypeText 判断类型文本是否为匿名 struct/union/class/enum 定义
// （typedef struct { ... } Point; 的底层文本）。
func isAnonymousTypeText(s string) bool {
	s = strings.TrimSpace(s)
	for _, kw := range []string{"struct", "union", "class", "enum"} {
		if !strings.HasPrefix(s, kw) {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(s, kw))
		return strings.HasPrefix(rest, "{") || strings.HasPrefix(rest, ":")
	}
	return false
}

// isIdentifier 判断是否为合法 C/C++ 标识符。
func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}
