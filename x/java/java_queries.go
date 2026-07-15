package java

// JavaActionQuery 定义了核心动作的 Tree-sitter 查询语句。
const JavaActionQuery = `
[
  ; ==========================================
  ; 1. Call 关系 (方法调用/方法引用)
  ; ==========================================
  (method_invocation name: (identifier) @call_target) @call_stmt
  (method_reference (identifier) @ref_target) @ref_stmt
  
  ; ==========================================
  ; 2. CREATE 关系 (对象/数组创建)
  ; ==========================================
  (object_creation_expression) @create_target
  (array_creation_expression) @create_target

  ; ==========================================
  ; 3. 显式构造函数调用 (super/this)
  ; ==========================================
  (explicit_constructor_invocation) @explicit_constructor_stmt

  ; ==========================================
  ; 4. Use 关系 (字段访问与变量读取) - 全量捞取，交由 Go 逻辑进行黑白名单拦截
  ; ==========================================
  (identifier) @id_atom
  (this) @id_atom

  ; ==========================================
  ; 5. Assign 关系 (赋值动作)
  ; ==========================================
  ; 统一捕获左值标识符
  (assignment_expression 
    left: [
        (identifier) @assign_target
        (field_access field: (identifier) @assign_target)
        (array_access array: (identifier) @assign_target)
    ])

  ; 匹配自增自减: count++, --count
  (update_expression 
    [
        (identifier) @assign_target
        (field_access field: (identifier) @assign_target)
    ])

  ; 增强 for 循环中的局部迭代变量赋值 (foreach variable)
  (enhanced_for_statement
    name: (identifier) @assign_target
  )

  ; 变量声明中的初始化: int a = 10;
  (variable_declarator 
    name: (identifier) @assign_target
    value: (_))

  ; ==========================================
  ; 6. Throw 关系 (抛出异常)
  ; ==========================================
  (throw_statement
    [
      (object_creation_expression) @throw_stmt
      (identifier) @throw_target
    ]
  ) @throw_stmt

  ; ==========================================
  ; 7. Cast 关系 (显式类型转换/类型检查与模式匹配)
  ; ==========================================
  (cast_expression) @cast_target
  (instanceof_expression) @cast_target
]
`
