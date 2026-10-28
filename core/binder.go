package core

// Binder 负责将元素上的 raw 类型字符串解析为 QualifiedName，建立全局符号索引。
// 该阶段串行执行（依赖全局索引一致性）。
type Binder interface {
	// BindSymbols 遍历全局上下文中的全部定义，完成类型绑定。
	BindSymbols(ctx *GlobalContext) error
}
