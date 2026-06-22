package constants

// =============================================================================
// java_collector 采集的 Element.Extra.Mores
// =============================================================================

const (
	BlockIsStatic                  = "java.block.is_static"            // 代码块是否为静态初始化块 (static { ... }) -> bool
	ClassIsStatic                  = "java.class.is_static"            // 类是否为静态内部类 -> bool
	ClassIsAbstract                = "java.class.is_abstract"          // 类是否为抽象类 (abstract) -> bool
	ClassIsFinal                   = "java.class.is_final"             // 类是否为终态类 (final) -> bool
	ClassSuperClass                = "java.class.superclass"           // 类的父类名称 (含泛型) -> string
	ClassImplementedInterfaces     = "java.class.interfaces"           // 类实现的接口列表 -> []string
	InterfaceImplementedInterfaces = "java.interface.interfaces"       // 接口继承的其他接口列表 -> []string
	MethodIsConstructor            = "java.method.is_constructor"      // 是否为构造方法 -> bool
	MethodIsDefault                = "java.method.is_default"          // 是否为接口中的默认方法 (default) -> bool
	MethodIsImplicit               = "java.method.is_implicit"         // 是否为隐式方法 (如编译器自动生成的构造函数) -> bool
	MethodIsAnnotation             = "java.method.is_annotation"       // 是否为注解类型中的元素方法 -> bool
	MethodComplexity               = "java.method.metrics.complexity"  // 方法圈复杂度 -> int
	MethodDefaultValue             = "java.method.default_value"       // 注解元素的默认值 -> string
	MethodReturnType               = "java.method.return_type"         // 方法返回值的原始类型文本 -> string
	MethodReturnTypeWithQN         = "java.method.return_type_with_qn" // 方法返回值类型QN -> string
	MethodParameters               = "java.method.parameters"          // 方法定义的参数列表 (含原始类型文本和名称) -> []string
	MethodParametersWithQN         = "java.method.parameters_with_qn"  // 方法定义的参数列表 (含类型QN和名称) -> []string
	MethodThrowsTypes              = "java.method.throws"              // 方法声明抛出的异常类型列表 (原始类型文本) -> []string
	MethodThrowsTypesWithQN        = "java.method.throws_with_qn"      // 方法声明抛出的异常类型列表 (类型QN) -> []string
	VariableRawType                = "java.variable.raw_type"          // 局部变量的类型文本 -> string
	VariableTypeWithQN             = "java.variable.type_with_qn"      // 局部变量的类型QN -> string
	VariableIsFinal                = "java.variable.is_final"          // 局部变量是否带有 final 修饰符 -> bool
	VariableIsParam                = "java.variable.is_param"          // 变量是否为方法参数 -> bool
	FieldRawType                   = "java.field.raw_type"             // 成员字段的类型文本 -> string
	FieldIsStatic                  = "java.field.is_static"            // 字段是否为静态 (static) -> bool
	FieldIsFinal                   = "java.field.is_final"             // 字段是否为终态 (final) -> bool
	FieldIsConstant                = "java.field.is_constant"          // 字段是否为常量 (通常指 static final 且有初始值) -> bool
	FieldIsRecordComponent         = "java.field.is_record_component"  // 字段是否为 Java Record 的组件成员 -> bool
	MethodRefReceiver              = "java.method_ref.receiver"        // 方法引用中的接收者 (如 System.out::println 中的 System.out) -> string
	MethodRefTarget                = "java.method_ref.target"          // 方法引用中的目标方法名 (如 println) -> string
	MethodRefTypeArgs              = "java.method_ref.type_arguments"  // 方法引用中显式指定的泛型参数 -> string
	EnumArguments                  = "java.enum.arguments"             // 枚举常量定义时传入的构造参数列表 -> []string
	LambdaParameters               = "java.lambda.parameters"          // Lambda 表达式的参数定义文本 -> string
	LambdaBodyIsBlock              = "java.lambda.is_block"            // Lambda 主体是否为大括号包裹的代码块 -> bool
	AnonymousClassType             = "java.anonymous_class.type"       // AnonymousClass 类型 -> string
	FileLOC                        = "java.file.metrics.loc"           // 文件有效行数 -> int
	FileRawLOC                     = "java.file.metrics.loc_raw"       // 文件行数 -> int
)

// Lombok 相关常量
const (
	MethodIsLombokGenerated = "java.method.is_lombok_generated" // 是否为Lombok生成的方法 -> bool
	LombokAnnotationType    = "java.lombok.annotation_type"     // Lombok注解类型 -> string
)
