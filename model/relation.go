package model

// DependencyType 表示依赖关系的类型。
type DependencyType string

const (
	// --- 1. 组织与逻辑层级关系 (Structural & Hierarchy) ---

	// Import 导入: 源码文件引用了外部包/类
	Import DependencyType = "IMPORT"

	// Export 导出: 模块或文件对外暴露的符号 (常见于 JS/TS, Go)
	Export DependencyType = "EXPORT"

	// Contain 包含: 物理或逻辑上的归属关系，用于构建全局符号树
	Contain DependencyType = "CONTAIN"

	// --- 2. 面向对象继承与实现 (Inheritance & Implementation) ---

	// Extend 继承: 类与类、接口与接口之间的继承
	Extend DependencyType = "EXTEND"

	// Implement 实现: 类实现接口
	Implement DependencyType = "IMPLEMENT"

	// Override 重写: 子类方法对父类/接口方法的具体实现或覆盖
	Override DependencyType = "OVERRIDE"

	// --- 3. 类型绑定与元数据 (Typing & Metadata) ---

	// Annotation 注解: 符号被特定注解修饰
	Annotation DependencyType = "ANNOTATION"

	// Parameter 参数类型: 方法/函数参数对类型的依赖
	Parameter DependencyType = "PARAMETER"

	// Return 返回类型: 方法返回值对类型的依赖
	Return DependencyType = "RETURN"

	// Throw 抛出异常: 方法声明抛出的异常类型
	Throw DependencyType = "THROW"

	// TypeArg 泛型实参: 变量定义中泛型参数指向的类型
	TypeArg DependencyType = "TYPE_ARG"

	// --- 4. 行为与执行流 (Behavioral & Execution) ---

	// Call 调用: 函数或方法的显式/隐式调用
	Call DependencyType = "CALL"

	// Create 显式实例创建: 源码中通过 new 关键字实例化对象
	Create DependencyType = "CREATE"

	// Cast 强转: 显式的类型转换
	Cast DependencyType = "CAST"

	// --- 5. 数据流与状态引用 (Data Flow & State) ---

	// Use 使用: 读取变量、字段或常量的值
	Use DependencyType = "USE"

	// Assign 赋值: 将值写入变量或字段
	Assign DependencyType = "ASSIGN"

	// Capture 捕获: 闭包（Lambda/匿名内部类）对外部局部变量的引用
	Capture DependencyType = "CAPTURE"

	// --- 6. 跨语言或特殊链接 (Special Linking) ---

	// ImplLink 符号链接: 用于 C/C++ 等区分声明与实现的语言，将原型链接到定义
	ImplLink DependencyType = "IMPL_LINK"

	// Mixin 混入: 动态将一个模块的功能注入另一个类
	Mixin DependencyType = "MIXIN"
)

// DependencyRelation 是工具的核心输出结构，描述了 Source 和 Target 之间的一个依赖关系。
type DependencyRelation struct {
	// Type: 依赖关系的类型 (e.g., CALL, IMPORT, EXTEND)
	Type DependencyType `json:"Type"`

	// Source: 关系的发起方（调用者、导入者、子类等）
	Source *CodeElement `json:"Source"`

	// Target: 关系的指向方（被调用者、被导入者、父类等）
	Target *CodeElement `json:"Target"`

	// Location: 关系发生的代码位置。
	Location *Location `json:"Location"`

	// Details: 存储关系的附加描述（如具体的调用签名、是否为静态调用等）
	Details string `json:"Details,omitempty"`

	// Mores: 预留的扩展属性字典
	Mores map[string]interface{} `json:"Mores,omitempty"`

	// ------------------------------------------------------------
	// 私有临时字段：仅语言实现内部各子阶段（extract→enricher）
	// 传递 AST 节点，引擎在流水线结束时统一清空，绝不进入 JSON 输出。
	// 值类型为 core/ast.Node 接口（语言无关）；model 保持零依赖，故用 any。
	// ------------------------------------------------------------
	tmpNode any // 触发该关系的 AST 节点
	tmpExpr any // 表达式节点
	tmpCtx  any // 上下文容器节点
}

// GetMores 惰性初始化并返回关系的 Mores 字典。
func (r *DependencyRelation) GetMores() map[string]interface{} {
	if r.Mores == nil {
		r.Mores = make(map[string]interface{})
	}
	return r.Mores
}

// --- 私有临时字段访问器（同包与语言实现通过导出方法访问） ---

// SetTmpNode 设置触发该关系的 AST 节点。
func (r *DependencyRelation) SetTmpNode(n any) { r.tmpNode = n }

// TmpNode 返回触发该关系的 AST 节点。
func (r *DependencyRelation) TmpNode() any { return r.tmpNode }

// SetTmpExpr 设置表达式节点。
func (r *DependencyRelation) SetTmpExpr(n any) { r.tmpExpr = n }

// TmpExpr 返回表达式节点。
func (r *DependencyRelation) TmpExpr() any { return r.tmpExpr }

// SetTmpCtx 设置上下文容器节点。
func (r *DependencyRelation) SetTmpCtx(n any) { r.tmpCtx = n }

// TmpCtx 返回上下文容器节点。
func (r *DependencyRelation) TmpCtx() any { return r.tmpCtx }

// ClearTemp 清空全部私有临时字段（流水线结束时调用，杜绝泄漏）。
func (r *DependencyRelation) ClearTemp() {
	r.tmpNode = nil
	r.tmpExpr = nil
	r.tmpCtx = nil
}
