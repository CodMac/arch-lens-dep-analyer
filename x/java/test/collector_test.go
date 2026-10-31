// Package test 收集层（第 3 层）单元测试：验证 Java Collector 对各语法结构的符号定义收集。
//
// 场景化组织结构（S1-S10）：
//
//	S1 基础声明   S2 类型声明   S3 成员   S4 嵌套结构
//	S5 作用域容器 S6 原生语法糖 S7 Lombok S8 度量
//	S9 泛型       S10 消歧盲区
//
// 断言值均来自 debug_dump 实测，覆盖 QN / Signature / Mores 三层。
package test

import (
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/core"
	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
)

// =============================================================================
// S1 基础声明：包 / 文件 / 类 / 字段 / 构造器 / 方法（collector/base + import）
// =============================================================================

func TestCollectorS1BaseDeclaration(t *testing.T) {
	gc := RunCollection(t, "x/java/testdata", ResolvePaths(t, "collector/base", "collector/import"))

	// --- 包元素 ---
	for _, pkg := range []string{"com", "com.example", "com.example.base", "com.example.base.test"} {
		AssertDef(t, gc, pkg, model.Package)
	}

	// --- 文件元素（QN 为绝对路径，按 kind+Name 定位）---
	userFile := findFile(t, gc, "User.java")
	AssertMores(t, gc, userFile.QualifiedName, constants.FileLOC, 45)
	AssertMores(t, gc, userFile.QualifiedName, constants.FileRawLOC, 63)

	// --- 类 User ---
	AssertDef(t, gc, "com.example.base.User", model.Class)
	AssertMores(t, gc, "com.example.base.User", constants.ClassIsAbstract, false)
	// 修饰符直接断言（public）
	AssertMods(t, gc, "com.example.base.User", "public")

	// --- 字段 ---
	AssertDef(t, gc, "com.example.base.User.id", model.Field)
	AssertMores(t, gc, "com.example.base.User.id", constants.FieldRawType, "String")
	AssertMores(t, gc, "com.example.base.User.id", constants.FieldIsStatic, false)
	AssertMods(t, gc, "com.example.base.User.id", "private")

	// 包私有字段 aInfo（无修饰符，raw_type 为嵌套类短名 AddonInfo）
	AssertDef(t, gc, "com.example.base.User.aInfo", model.Field)
	AssertMores(t, gc, "com.example.base.User.aInfo", constants.FieldRawType, "AddonInfo")
	AssertMods(t, gc, "com.example.base.User.aInfo") // 空期望：包私有（无 public/private/protected）

	// 静态常量字段（is_constant=true）
	AssertDef(t, gc, "com.example.base.User.DEFAULT_ID", model.Field)
	AssertMores(t, gc, "com.example.base.User.DEFAULT_ID", constants.FieldIsStatic, true)
	AssertMores(t, gc, "com.example.base.User.DEFAULT_ID", constants.FieldIsFinal, true)
	AssertMores(t, gc, "com.example.base.User.DEFAULT_ID", constants.FieldIsConstant, true)
	AssertMods(t, gc, "com.example.base.User.DEFAULT_ID", "private", "static", "final")

	// --- 构造器（QN 含参数列表）---
	AssertDef(t, gc, "com.example.base.User.User(String)", model.Method)
	AssertMores(t, gc, "com.example.base.User.User(String)", constants.MethodIsConstructor, true)
	AssertMores(t, gc, "com.example.base.User.User(String)", constants.MethodParameters, []string{"String username"})
	// 构造器参数变量（is_param=true）
	AssertDef(t, gc, "com.example.base.User.User(String).username", model.Variable)
	AssertMores(t, gc, "com.example.base.User.User(String).username", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.User.User(String).username", constants.VariableRawType, "String")

	// --- 方法 ---
	AssertDef(t, gc, "com.example.base.User.getId()", model.Method)
	AssertMores(t, gc, "com.example.base.User.getId()", constants.MethodReturnType, "String")
	AssertMores(t, gc, "com.example.base.User.setUsername(String)", constants.MethodReturnType, "void")
	// 方法参数变量（is_param=true）
	AssertDef(t, gc, "com.example.base.User.setUsername(String).username", model.Variable)
	AssertMores(t, gc, "com.example.base.User.setUsername(String).username", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.User.setUsername(String).username", constants.VariableRawType, "String")

	// --- 嵌套类 AddonInfo ---
	AssertDef(t, gc, "com.example.base.User.AddonInfo", model.Class)
	AssertMores(t, gc, "com.example.base.User.AddonInfo", constants.ClassIsStatic, true)
	AssertMods(t, gc, "com.example.base.User.AddonInfo", "public", "static")
	AssertDef(t, gc, "com.example.base.User.AddonInfo.age", model.Field)
	AssertMores(t, gc, "com.example.base.User.AddonInfo.age", constants.FieldRawType, "int")
	AssertMods(t, gc, "com.example.base.User.AddonInfo.age", "private")
	AssertDef(t, gc, "com.example.base.User.AddonInfo.chooseUnit(long)", model.Method)
	AssertMods(t, gc, "com.example.base.User.AddonInfo.chooseUnit(long)", "private", "static")
	// 全限定类型字段（raw_type 保留全限定名）+ protected 修饰符
	AssertMores(t, gc, "com.example.base.User.AddonInfo.birthday", constants.FieldRawType, "java.util.Date")
	AssertMods(t, gc, "com.example.base.User.AddonInfo.birthday", "protected")

	// --- 泛型抽象类 AbstractBaseEntity<ID> ---
	AssertDef(t, gc, "com.example.base.AbstractBaseEntity", model.Class)
	AssertMores(t, gc, "com.example.base.AbstractBaseEntity", constants.ClassIsAbstract, true)
	AssertMores(t, gc, "com.example.base.AbstractBaseEntity", constants.ClassTypeParameters, "<ID>")
	AssertMods(t, gc, "com.example.base.AbstractBaseEntity", "public", "abstract")
	// 泛型类型参数方法/字段（ID 类型参数贯穿）
	AssertMores(t, gc, "com.example.base.AbstractBaseEntity.getId()", constants.MethodReturnType, "ID")
	AssertMores(t, gc, "com.example.base.AbstractBaseEntity.id", constants.FieldRawType, "ID")
	AssertMods(t, gc, "com.example.base.AbstractBaseEntity.id", "protected")
	// 外部 import 引入的字段类型（Date 短名 -> raw_type 保留短名，import 表承载 QN 升级）
	AssertMores(t, gc, "com.example.base.AbstractBaseEntity.createdAt", constants.FieldRawType, "Date")
	AssertMods(t, gc, "com.example.base.AbstractBaseEntity.createdAt", "private")

	// --- 泛型父类 + 接口（UserServiceImpl）---
	AssertMores(t, gc, "com.example.base.UserServiceImpl", constants.ClassSuperClass, "AbstractBaseEntity<String>")
	AssertMores(t, gc, "com.example.base.UserServiceImpl", constants.ClassImplementedInterfaces, []string{"DataProcessor<AbstractBaseEntity<String>>"})
	AssertMores(t, gc, "com.example.base.UserServiceImpl.processAll(String)", constants.MethodReturnType, "List<AbstractBaseEntity<String>>")
	AssertMores(t, gc, "com.example.base.UserServiceImpl.processAll(String)", constants.MethodThrowsTypes, []string{"RuntimeException"})
	AssertMores(t, gc, "com.example.base.UserServiceImpl.close()", constants.MethodThrowsTypes, []string{"Exception"})
	// 类级注解（@Loggable）与方法级注解（@Override）
	AssertAnnos(t, gc, "com.example.base.UserServiceImpl", "@Loggable")
	AssertAnnos(t, gc, "com.example.base.UserServiceImpl.processAll(String)", "@Override")

	// --- 接口 DataProcessor（泛型上界 + 参数变量 + default 方法）---
	AssertMores(t, gc, "com.example.base.DataProcessor", constants.ClassTypeParameters, "<T extends AbstractBaseEntity<?>>")
	AssertMores(t, gc, "com.example.base.DataProcessor.processAll(String).batchId", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.DataProcessor.processAll(String).batchId", constants.VariableRawType, "String")
	// default 方法：default 修饰符进入 Modifiers，且 is_default Mores 为 true
	AssertMods(t, gc, "com.example.base.DataProcessor.stop()", "default")
	AssertMores(t, gc, "com.example.base.DataProcessor.stop()", constants.MethodIsDefault, true)
	AssertMores(t, gc, "com.example.base.DataProcessor.stop()", constants.MethodReturnType, "void")

	// --- CallbackManager（局部类宿主方法 + 匿名类载体局部变量）---
	AssertDef(t, gc, "com.example.base.CallbackManager.register()", model.Method)
	AssertMods(t, gc, "com.example.base.CallbackManager.register()", "public")
	AssertMores(t, gc, "com.example.base.CallbackManager.register()", constants.MethodReturnType, "void")
	AssertDef(t, gc, "com.example.base.CallbackManager.register().r", model.Variable)
	AssertMores(t, gc, "com.example.base.CallbackManager.register().r", constants.VariableRawType, "Runnable")

	// --- ConfigService（变长参数/多维数组参数变量 + 复杂注解）---
	AssertMores(t, gc, "com.example.base.ConfigService.updateConfigs(String[],Object...).keys", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.ConfigService.updateConfigs(String[],Object...).keys", constants.VariableRawType, "String[]")
	AssertMores(t, gc, "com.example.base.ConfigService.updateConfigs(String[],Object...).values", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.ConfigService.updateConfigs(String[],Object...).values", constants.VariableRawType, "Object...")
	// 复杂注解：数组值 @SuppressWarnings + 具名参数 @Deprecated
	AssertAnnos(t, gc, "com.example.base.ConfigService.legacyMethod()",
		`@SuppressWarnings({"unchecked", "rawtypes"})`, `@Deprecated(since = "1.2", forRemoval = true)`)

	// --- ImportTest（collector/import：单类型/通配符/静态导入）---
	AssertDef(t, gc, "com.example.base.test.ImportTest", model.Class)
	AssertMores(t, gc, "com.example.base.test.ImportTest.list", constants.FieldRawType, "List<String>")
	// --- Imports 表：类型导入 / 通配符导入 / 静态通配符导入 / 静态导入的识别 ---
	assertImports(t, gc, "collector/import/ImportTest.java", []ImportWant{
		{Alias: "List", RawPath: "java.util.List", Kind: model.Class},
		{Alias: "*", RawPath: "java.util.*", Kind: model.Package, Wildcard: true},
		{Alias: "*", RawPath: "java.lang.Math.*", Kind: model.Constant, Static: true, Wildcard: true},
		{Alias: "sort", RawPath: "java.util.Collections.sort", Kind: model.Constant, Static: true},
	})
}

// ImportWant 描述一条预期的导入条目（用于 Imports 表断言）。
type ImportWant struct {
	Alias    string
	RawPath  string
	Kind     model.ElementKind
	Static   bool
	Wildcard bool
}

// assertImports 断言某文件 Imports 表包含全部预期条目（顺序无关）。
// relPath 使用正斜杠分隔（如 "collector/import/ImportTest.java"），跨平台安全。
func assertImports(t *testing.T, gc *core.GlobalContext, relPath string, wants []ImportWant) {
	t.Helper()
	fc, ok := findFileCtx(gc, relPath)
	if !ok {
		t.Fatalf("file context not found: %s", relPath)
	}
	matched := make(map[string]bool, len(wants))
	for _, w := range wants {
		found := false
		for _, info := range fc.Imports[w.Alias] {
			if info.RawImportPath == w.RawPath && info.Kind == w.Kind && info.IsStatic == w.Static && info.IsWildcard == w.Wildcard {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s: import alias=%s path=%s kind=%s static=%v wildcard=%v NOT FOUND",
				relPath, w.Alias, w.RawPath, w.Kind, w.Static, w.Wildcard)
			continue
		}
		matched[w.Alias+"|"+w.RawPath] = true
	}
	// 反向校验：Imports 表不包含多余条目。
	for alias, infos := range fc.Imports {
		for _, info := range infos {
			key := alias + "|" + info.RawImportPath
			if !matched[key] {
				t.Errorf("%s: unexpected import alias=%s path=%s kind=%s static=%v wildcard=%v",
					relPath, alias, info.RawImportPath, info.Kind, info.IsStatic, info.IsWildcard)
			}
		}
	}
}

// findFileCtx 按相对路径（正斜杠）查找文件上下文，兼容 Windows 反斜杠分隔。
func findFileCtx(gc *core.GlobalContext, relPath string) (*core.FileContext, bool) {
	want := filepath.ToSlash(relPath)
	for key, fc := range gc.FileContexts() {
		if filepath.ToSlash(key) == want {
			return fc, true
		}
	}
	return nil, false
}

// findFile 在全局上下文中按文件名查找 FILE 元素。
func findFile(t *testing.T, gc *core.GlobalContext, name string) *model.CodeElement {
	t.Helper()
	for _, def := range gc.Definitions() {
		if def.Element.Kind == model.File && def.Element.Name == name {
			return def.Element
		}
	}
	t.Fatalf("file element %s not found", name)
	return nil
}

// =============================================================================
// S2 类型声明：抽象类 / 终态类 / 接口 / 注解 / 枚举（collector/base + annotation + enum）
// =============================================================================

func TestCollectorS2TypeDeclaration(t *testing.T) {
	gc := RunCollection(t, "x/java/testdata", ResolvePaths(t, "collector/base", "collector/annotation", "collector/enum"))

	// --- 抽象类 BaseClass ---
	AssertDef(t, gc, "com.example.base.BaseClass", model.Class)
	AssertMores(t, gc, "com.example.base.BaseClass", constants.ClassIsAbstract, true)
	AssertMores(t, gc, "com.example.base.BaseClass", constants.ClassImplementedInterfaces, []string{"Serializable"})
	// 类级注解（含带参注解 @SuppressWarnings）与修饰符
	AssertAnnos(t, gc, "com.example.base.BaseClass", "@Deprecated", `@SuppressWarnings("unused")`)
	AssertMods(t, gc, "com.example.base.BaseClass", "public", "abstract")

	// --- 终态类 FinalClass（继承 + 多接口）---
	AssertDef(t, gc, "com.example.base.FinalClass", model.Class)
	AssertMores(t, gc, "com.example.base.FinalClass", constants.ClassIsFinal, true)
	AssertMores(t, gc, "com.example.base.FinalClass", constants.ClassSuperClass, "BaseClass")
	AssertMores(t, gc, "com.example.base.FinalClass", constants.ClassImplementedInterfaces, []string{"Cloneable", "Runnable"})
	// 包私有 final 类（无 public）+ override 方法注解
	AssertMods(t, gc, "com.example.base.FinalClass", "final")
	AssertAnnos(t, gc, "com.example.base.FinalClass.run()", "@Override")

	// --- 接口 DataProcessor（继承接口 + throws + 泛型）---
	AssertDef(t, gc, "com.example.base.DataProcessor", model.Interface)
	AssertMores(t, gc, "com.example.base.DataProcessor", constants.InterfaceImplementedInterfaces, []string{"Runnable", "AutoCloseable"})
	AssertMores(t, gc, "com.example.base.DataProcessor.processAll(String)", constants.MethodReturnType, "List<T>")
	AssertMores(t, gc, "com.example.base.DataProcessor.processAll(String)", constants.MethodThrowsTypes, []string{"RuntimeException", "Exception"})
	AssertMores(t, gc, "com.example.base.DataProcessor.processAll(String)", constants.MethodParameters, []string{"String batchId"})

	// --- 注解 Loggable（元素方法 + 默认值 + 元注解）---
	AssertDef(t, gc, "com.example.base.annotation.Loggable", model.KAnnotation)
	AssertMods(t, gc, "com.example.base.annotation.Loggable", "public")
	// 元注解：@Retention（枚举参数）+ @Target（数组值）
	AssertAnnos(t, gc, "com.example.base.annotation.Loggable", `@Retention(RetentionPolicy.RUNTIME)`, `@Target({ElementType.TYPE, ElementType.METHOD})`)
	AssertMores(t, gc, "com.example.base.annotation.Loggable.level()", constants.MethodIsAnnotation, true)
	AssertMores(t, gc, "com.example.base.annotation.Loggable.level()", constants.MethodDefaultValue, `"INFO"`)
	AssertMores(t, gc, "com.example.base.annotation.Loggable.trace()", constants.MethodIsAnnotation, true)
	AssertMores(t, gc, "com.example.base.annotation.Loggable.trace()", constants.MethodDefaultValue, "false")

	// --- 枚举 ErrorCode（常量 + 字段 + 构造器 + 方法 + 隐式方法）---
	AssertDef(t, gc, "com.example.base.enum.ErrorCode", model.Enum)
	AssertMods(t, gc, "com.example.base.enum.ErrorCode", "public")

	// 枚举常量（带构造参数）
	AssertDef(t, gc, "com.example.base.enum.ErrorCode.USER_NOT_FOUND", model.EnumConstant)
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.USER_NOT_FOUND", constants.EnumArguments, []string{"404", `"User not found in repository"`})
	AssertDef(t, gc, "com.example.base.enum.ErrorCode.NAME_EMPTY", model.EnumConstant)
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.NAME_EMPTY", constants.EnumArguments, []string{"400", `"User name cannot be empty"`})

	// 枚举字段（final，private final 修饰符）
	AssertDef(t, gc, "com.example.base.enum.ErrorCode.code", model.Field)
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.code", constants.FieldRawType, "int")
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.code", constants.FieldIsFinal, true)
	AssertMods(t, gc, "com.example.base.enum.ErrorCode.code", "private", "final")
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.code", constants.FieldIsConstant, false)
	AssertDef(t, gc, "com.example.base.enum.ErrorCode.message", model.Field)
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.message", constants.FieldRawType, "String")
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.message", constants.FieldIsFinal, true)
	AssertMods(t, gc, "com.example.base.enum.ErrorCode.message", "private", "final")

	// 枚举构造器（包私有）
	AssertDef(t, gc, "com.example.base.enum.ErrorCode.ErrorCode(int,String)", model.Method)
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.ErrorCode(int,String)", constants.MethodIsConstructor, true)
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.ErrorCode(int,String)", constants.MethodParameters, []string{"int code", "String message"})
	AssertMods(t, gc, "com.example.base.enum.ErrorCode.ErrorCode(int,String)") // 包私有构造器
	// 构造器参数变量
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.ErrorCode(int,String).code", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.ErrorCode(int,String).code", constants.VariableRawType, "int")
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.ErrorCode(int,String).message", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.ErrorCode(int,String).message", constants.VariableRawType, "String")

	// 枚举方法
	AssertDef(t, gc, "com.example.base.enum.ErrorCode.getCode()", model.Method)
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.getCode()", constants.MethodReturnType, "int")
	AssertDef(t, gc, "com.example.base.enum.ErrorCode.getMessage()", model.Method)
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.getMessage()", constants.MethodReturnType, "String")
	AssertMods(t, gc, "com.example.base.enum.ErrorCode.getMessage()", "public")

	// 隐式合成方法（values / valueOf）
	AssertDef(t, gc, "com.example.base.enum.ErrorCode.values()", model.Method)
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.values()", constants.MethodIsImplicit, true)
	AssertDef(t, gc, "com.example.base.enum.ErrorCode.valueOf(String)", model.Method)
	AssertMores(t, gc, "com.example.base.enum.ErrorCode.valueOf(String)", constants.MethodIsImplicit, true)

	// --- 枚举 + Lombok @Getter（EnumLombok）：枚举内静态块访问自身 getter 需命中生成方法 ---
	AssertDef(t, gc, "com.example.base.enum.EnumLombok", model.Enum)
	// @Getter 为非 static 字段生成 getter（枚举字段 private final 也生成）
	AssertDef(t, gc, "com.example.base.enum.EnumLombok.getKey()", model.Method)
	AssertMores(t, gc, "com.example.base.enum.EnumLombok.getKey()", constants.MethodIsLombokGenerated, true)
	AssertMores(t, gc, "com.example.base.enum.EnumLombok.getKey()", constants.MethodReturnType, "String")
	AssertDef(t, gc, "com.example.base.enum.EnumLombok.getCode()", model.Method)
	AssertMores(t, gc, "com.example.base.enum.EnumLombok.getCode()", constants.MethodIsLombokGenerated, true)
	AssertMores(t, gc, "com.example.base.enum.EnumLombok.getCode()", constants.MethodReturnType, "int")
	// 枚举常量不是 Field，不应被生成 getter
	AssertNoDef(t, gc, "com.example.base.enum.EnumLombok.getFIRST()")
	// 隐式方法仍生成
	AssertDef(t, gc, "com.example.base.enum.EnumLombok.values()", model.Method)
	AssertDef(t, gc, "com.example.base.enum.EnumLombok.valueOf(String)", model.Method)
}

// =============================================================================
// S3 成员：方法重载 / 变长参数 / 多维数组 / 构造器识别（collector/method + parameter + base）
// =============================================================================

func TestCollectorS3Member(t *testing.T) {
	gc := RunCollection(t, "x/java/testdata", ResolvePaths(t, "collector/method", "collector/parameter", "collector/base"))

	// --- 方法重载（QN 以参数类型区分）---
	AssertDef(t, gc, "com.example.base.MethodTest.exec(int)", model.Method)
	AssertDef(t, gc, "com.example.base.MethodTest.exec(String)", model.Method)
	AssertDef(t, gc, "com.example.base.MethodTest.exec(int,String)", model.Method)
	AssertMores(t, gc, "com.example.base.MethodTest.exec(int,String)", constants.MethodParameters, []string{"int i", "String s"})
	AssertMores(t, gc, "com.example.base.MethodTest.exec(String)", constants.MethodReturnType, "String")
	// 重载方法各自的参数列表与参数变量（QN 以参数类型区分）
	AssertMores(t, gc, "com.example.base.MethodTest.exec(int)", constants.MethodParameters, []string{"int i"})
	AssertMores(t, gc, "com.example.base.MethodTest.exec(String)", constants.MethodParameters, []string{"String s"})
	AssertMores(t, gc, "com.example.base.MethodTest.exec(int).i", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.MethodTest.exec(int).i", constants.VariableRawType, "int")
	AssertMores(t, gc, "com.example.base.MethodTest.exec(String).s", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.MethodTest.exec(String).s", constants.VariableRawType, "String")
	AssertMores(t, gc, "com.example.base.MethodTest.exec(int,String).i", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.MethodTest.exec(int,String).s", constants.VariableIsParam, true)
	// 重载方法返回类型与复杂度（void 返回 / 各重载独立计复杂度）
	AssertMores(t, gc, "com.example.base.MethodTest.exec(int)", constants.MethodReturnType, "void")
	AssertMores(t, gc, "com.example.base.MethodTest.exec(int,String)", constants.MethodReturnType, "void")
	AssertMores(t, gc, "com.example.base.MethodTest.exec(int)", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.base.MethodTest.exec(String)", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.base.MethodTest.exec(int,String)", constants.MethodComplexity, 1)

	// 显式构造器（public 修饰符 + is_constructor；无 is_implicit 键=非隐式合成）
	AssertDef(t, gc, "com.example.base.MethodTest.MethodTest()", model.Method)
	AssertMores(t, gc, "com.example.base.MethodTest.MethodTest()", constants.MethodIsConstructor, true)
	AssertMods(t, gc, "com.example.base.MethodTest.MethodTest()", "public")

	// --- 变长参数 + 多维数组（ConfigService）---
	AssertDef(t, gc, "com.example.base.ConfigService.updateConfigs(String[],Object...)", model.Method)
	AssertMores(t, gc, "com.example.base.ConfigService.updateConfigs(String[],Object...)", constants.MethodParameters, []string{"String[] keys", "Object... values"})
	// 数组/变长参数变量：[] 与 ... 穿透到变量 raw_type
	AssertMores(t, gc, "com.example.base.ConfigService.updateConfigs(String[],Object...).keys", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.ConfigService.updateConfigs(String[],Object...).keys", constants.VariableRawType, "String[]")
	AssertMores(t, gc, "com.example.base.ConfigService.updateConfigs(String[],Object...).values", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.ConfigService.updateConfigs(String[],Object...).values", constants.VariableRawType, "Object...")
	// 变长参数方法返回类型/复杂度
	AssertMores(t, gc, "com.example.base.ConfigService.updateConfigs(String[],Object...)", constants.MethodReturnType, "void")
	AssertMores(t, gc, "com.example.base.ConfigService.updateConfigs(String[],Object...)", constants.MethodComplexity, 1)

	// --- 参数作用域：构造器 / 变长参数 / 内部类参数（ParameterScopeTest）---
	// 构造器名与类名一致（ParameterScopeTest 而非 ParameterScopeTester）
	AssertDef(t, gc, "com.example.base.test.ParameterScopeTest.ParameterScopeTest(String)", model.Method)
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.ParameterScopeTest(String)", constants.MethodIsConstructor, true)
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.ParameterScopeTest(String)", constants.MethodParameters, []string{"String initialConfig"})
	// 构造器参数变量
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.ParameterScopeTest(String).initialConfig", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.ParameterScopeTest(String).initialConfig", constants.VariableRawType, "String")

	// 变长参数方法（QN 含 ...）
	AssertDef(t, gc, "com.example.base.test.ParameterScopeTest.execute(int,String...)", model.Method)
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.execute(int,String...)", constants.MethodParameters, []string{"int times", "String... labels"})
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.execute(int,String...)", constants.MethodReturnType, "void")
	// 变长参数变量：... 穿透到变量 raw_type
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.execute(int,String...).times", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.execute(int,String...).times", constants.VariableRawType, "int")
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.execute(int,String...).labels", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.execute(int,String...).labels", constants.VariableRawType, "String...")

	// 内部类方法参数
	AssertDef(t, gc, "com.example.base.test.ParameterScopeTest.InnerWorker", model.Class)
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.InnerWorker", constants.ClassIsStatic, false)
	AssertDef(t, gc, "com.example.base.test.ParameterScopeTest.InnerWorker.doWork(long)", model.Method)
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.InnerWorker.doWork(long)", constants.MethodParameters, []string{"long duration"})
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.InnerWorker.doWork(long)", constants.MethodReturnType, "void")
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.InnerWorker.doWork(long)", constants.MethodComplexity, 1)
	// 内部类参数变量 + 内部类隐式构造器
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.InnerWorker.doWork(long).duration", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.InnerWorker.doWork(long).duration", constants.VariableRawType, "long")
	AssertDef(t, gc, "com.example.base.test.ParameterScopeTest.InnerWorker.InnerWorker()", model.Method)
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.InnerWorker.InnerWorker()", constants.MethodIsConstructor, true)
	AssertMores(t, gc, "com.example.base.test.ParameterScopeTest.InnerWorker.InnerWorker()", constants.MethodIsImplicit, true)
}

// =============================================================================
// S5 作用域容器：局部变量作用域重定位 / 块 / lambda（collector/var_scope）
// =============================================================================

func TestCollectorS5ScopeContainer(t *testing.T) {
	gc := RunCollection(t, "x/java/testdata", ResolvePaths(t, "collector/var_scope", "extractor/throw"))

	// --- 基础方法作用域变量 ---
	AssertDef(t, gc, "com.example.base.test.ScopeVariableTest.test().list", model.Variable)
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().list", constants.VariableRawType, "List<String>")

	// --- try-with-resources 资源变量（块重定位到 block$1，is_param=false）---
	AssertDef(t, gc, "com.example.base.test.ScopeVariableTest.test().block$1.bis", model.Variable)
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().block$1.bis", constants.VariableRawType, "BufferedInputStream")
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().block$1.bis", constants.VariableIsParam, false)

	// --- catch 参数（块重定位到 block$2，类型取自 catch_type）---
	AssertDef(t, gc, "com.example.base.test.ScopeVariableTest.test().block$2.e", model.Variable)
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().block$2.e", constants.VariableRawType, "Exception")
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().block$2.e", constants.VariableIsParam, false)

	// --- 多异常联合类型 catch 参数（catch (IOException | SQLException e)，取第一个类型）---
	AssertDef(t, gc, "com.example.rel.ThrowRelationSuite.rethrowMulti().block$2.e", model.Variable)
	AssertMores(t, gc, "com.example.rel.ThrowRelationSuite.rethrowMulti().block$2.e", constants.VariableRawType, "IOException")
	AssertMores(t, gc, "com.example.rel.ThrowRelationSuite.rethrowMulti().block$2.e", constants.VariableIsParam, false)

	// --- for 循环变量（block$3）---
	AssertDef(t, gc, "com.example.base.test.ScopeVariableTest.test().block$3.i", model.Variable)
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().block$3.i", constants.VariableRawType, "int")

	// --- 增强 for 变量（block$4）---
	AssertDef(t, gc, "com.example.base.test.ScopeVariableTest.test().block$4.item", model.Variable)
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().block$4.item", constants.VariableRawType, "String")

	// --- instanceof 模式变量（block$5，类型 unknown）---
	AssertDef(t, gc, "com.example.base.test.ScopeVariableTest.test().block$5.s", model.Variable)
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().block$5.s", constants.VariableRawType, "unknown")

	// --- 方法内局部变量 ---
	AssertDef(t, gc, "com.example.base.test.ScopeVariableTest.test().obj", model.Variable)
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().obj", constants.VariableRawType, "Object")
	// 宿主方法（复杂度含 try/catch + for + enhanced-for + instanceof 判定）
	AssertDef(t, gc, "com.example.base.test.ScopeVariableTest.test()", model.Method)
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test()", constants.MethodReturnType, "void")
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test()", constants.MethodComplexity, 5)
	// 各作用域块自身复杂度（独立计算，均为 1）
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().block$1", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().block$2", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().block$3", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().block$4", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.base.test.ScopeVariableTest.test().block$5", constants.MethodComplexity, 1)

	// --- if/else 块变量作用域重定位（Bug A 回归：else 块变量须挂 else block，非 if 体块）---
	// tree-sitter 中 if/else 是同一 if_statement 下两个 block 子节点，旧逻辑固定匹配第一个
	// block 导致 else 块变量被误挂到 if 体块。修复后 ifVar->block$1、elseVar->block$2。
	AssertDef(t, gc, "com.example.base.test.IfElseScopeTest.test(int).block$1.ifVar", model.Variable)
	AssertMores(t, gc, "com.example.base.test.IfElseScopeTest.test(int).block$1.ifVar", constants.VariableRawType, "int")
	AssertDef(t, gc, "com.example.base.test.IfElseScopeTest.test(int).block$2.elseVar", model.Variable)
	AssertMores(t, gc, "com.example.base.test.IfElseScopeTest.test(int).block$2.elseVar", constants.VariableRawType, "int")
	// 宿主方法与各块复杂度
	AssertDef(t, gc, "com.example.base.test.IfElseScopeTest.test(int)", model.Method)
	AssertMores(t, gc, "com.example.base.test.IfElseScopeTest.test(int)", constants.MethodComplexity, 2)
	AssertMores(t, gc, "com.example.base.test.IfElseScopeTest.test(int).block$1", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.base.test.IfElseScopeTest.test(int).block$2", constants.MethodComplexity, 1)

	// --- 增强 for 循环体内变量归 for 体块（Bug A 延伸：findNearestBlockParent 缺 enhanced_for 分支）---
	// 旧逻辑向上命中外层 if_statement，for 体内变量被误挂 if 体块 block$1；
	// 修复后 loopVar/item 正确归 for 体块 block$1.block$1。
	AssertDef(t, gc, "com.example.base.test.IfElseScopeTest.forEach(int).block$1.block$1.loopVar", model.Variable)
	AssertMores(t, gc, "com.example.base.test.IfElseScopeTest.forEach(int).block$1.block$1.loopVar", constants.VariableRawType, "int")
	AssertDef(t, gc, "com.example.base.test.IfElseScopeTest.forEach(int).block$1.block$1.item", model.Variable)
	AssertMores(t, gc, "com.example.base.test.IfElseScopeTest.forEach(int).block$1.block$1.item", constants.VariableRawType, "String")
	// 宿主方法复杂度：if(1) + enhanced-for(1) + 基线 = 3；if 体块含 enhanced-for = 2；for 体块 = 1
	AssertDef(t, gc, "com.example.base.test.IfElseScopeTest.forEach(int)", model.Method)
	AssertMores(t, gc, "com.example.base.test.IfElseScopeTest.forEach(int)", constants.MethodComplexity, 3)
	AssertMores(t, gc, "com.example.base.test.IfElseScopeTest.forEach(int).block$1", constants.MethodComplexity, 2)
	AssertMores(t, gc, "com.example.base.test.IfElseScopeTest.forEach(int).block$1.block$1", constants.MethodComplexity, 1)

	// --- 同名变量在不同容器（不触发重名消歧）---
	AssertDef(t, gc, "com.example.base.ScopeTest.test().x", model.Variable)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().x", constants.VariableRawType, "int")
	AssertDef(t, gc, "com.example.base.ScopeTest.test().block$1.x", model.Variable)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().block$1.x", constants.VariableRawType, "int")
	AssertDef(t, gc, "com.example.base.ScopeTest.test().block$2.x", model.Variable)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().block$2.x", constants.VariableRawType, "int")
	// 宿主方法与各块复杂度
	AssertDef(t, gc, "com.example.base.ScopeTest.test()", model.Method)
	AssertMores(t, gc, "com.example.base.ScopeTest.test()", constants.MethodComplexity, 2)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().block$1", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().block$2", constants.MethodComplexity, 1)

	// --- Lambda：块 lambda 与表达式 lambda ---
	// lambda$1：() -> { x/a/b/c }（块 lambda）
	AssertDef(t, gc, "com.example.base.ScopeTest.test().lambda$1", model.Lambda)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$1", constants.LambdaParameters, "()")
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$1", constants.LambdaBodyIsBlock, true)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$1", constants.MethodComplexity, 1)
	AssertDef(t, gc, "com.example.base.ScopeTest.test().lambda$1.a", model.Variable)
	// 多变量声明 a, b, c 全部进入 lambda 作用域（is_param=false）
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$1.a", constants.VariableRawType, "int")
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$1.a", constants.VariableIsParam, false)
	AssertDef(t, gc, "com.example.base.ScopeTest.test().lambda$1.b", model.Variable)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$1.b", constants.VariableRawType, "int")
	AssertDef(t, gc, "com.example.base.ScopeTest.test().lambda$1.c", model.Variable)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$1.c", constants.VariableRawType, "int")
	// lambda 作用域内同名 x（与方法体 block$N.x 各自独立计数）
	AssertDef(t, gc, "com.example.base.ScopeTest.test().lambda$1.x", model.Variable)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$1.x", constants.VariableRawType, "int")

	// lambda$2：(p1, p2) -> {...}（推断类型参数：is_param=false + raw_type=inferred）
	AssertDef(t, gc, "com.example.base.ScopeTest.test().lambda$2", model.Lambda)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$2", constants.LambdaParameters, "(p1, p2)")
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$2", constants.LambdaBodyIsBlock, true)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$2", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$2.p1", constants.VariableRawType, "inferred")
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$2.p1", constants.VariableIsParam, false)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$2.p2", constants.VariableRawType, "inferred")
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$2.p2", constants.VariableIsParam, false)
	AssertDef(t, gc, "com.example.base.ScopeTest.test().lambda$2.sum", model.Variable)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$2.sum", constants.VariableRawType, "int")

	// lambda$3：(int v1, int v2) -> expr（表达式 lambda，参数带类型：is_param=true）
	AssertDef(t, gc, "com.example.base.ScopeTest.test().lambda$3", model.Lambda)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$3", constants.LambdaParameters, "(int v1, int v2)")
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$3", constants.LambdaBodyIsBlock, false)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$3", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$3.v1", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$3.v1", constants.VariableRawType, "int")
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$3.v2", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$3.v2", constants.VariableRawType, "int")

	// lambda$4：s -> {...}（推断类型参数：is_param=false）
	AssertDef(t, gc, "com.example.base.ScopeTest.test().lambda$4", model.Lambda)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$4", constants.LambdaParameters, "s")
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$4", constants.LambdaBodyIsBlock, true)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$4", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$4.s", constants.VariableRawType, "inferred")
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$4.s", constants.VariableIsParam, false)
	AssertDef(t, gc, "com.example.base.ScopeTest.test().lambda$4.prefix", model.Variable)
	AssertMores(t, gc, "com.example.base.ScopeTest.test().lambda$4.prefix", constants.VariableRawType, "String")
}

// =============================================================================
// S4 嵌套结构：静态/实例初始化块 / 内部类 / 局部类 / 匿名类（collector/class + base）
// =============================================================================

func TestCollectorS4NestedStructure(t *testing.T) {
	gc := RunCollection(t, "x/java/testdata", ResolvePaths(t, "collector/class", "collector/base"))

	// --- OuterClass：静态/实例初始化块 ---
	AssertDef(t, gc, "com.example.base.OuterClass", model.Class)
	AssertDef(t, gc, "com.example.base.OuterClass.$static$1", model.ScopeBlock)
	AssertMores(t, gc, "com.example.base.OuterClass.$static$1", constants.BlockIsStatic, true)
	AssertMores(t, gc, "com.example.base.OuterClass.$static$1", constants.MethodComplexity, 1)
	AssertDef(t, gc, "com.example.base.OuterClass.$instance$1", model.ScopeBlock)
	AssertMores(t, gc, "com.example.base.OuterClass.$instance$1", constants.BlockIsStatic, false)
	AssertMores(t, gc, "com.example.base.OuterClass.$instance$1", constants.MethodComplexity, 1)

	// 静态字段（public static，非 final 非常量）
	AssertDef(t, gc, "com.example.base.OuterClass.count", model.Field)
	AssertMores(t, gc, "com.example.base.OuterClass.count", constants.FieldIsStatic, true)
	AssertMores(t, gc, "com.example.base.OuterClass.count", constants.FieldIsConstant, false)
	AssertMods(t, gc, "com.example.base.OuterClass.count", "public", "static")

	// --- 内部类 / 静态嵌套类 ---
	AssertDef(t, gc, "com.example.base.OuterClass.InnerClass", model.Class)
	AssertMores(t, gc, "com.example.base.OuterClass.InnerClass", constants.ClassIsStatic, false)
	AssertMores(t, gc, "com.example.base.OuterClass.InnerClass", constants.ClassIsAbstract, false)
	AssertMores(t, gc, "com.example.base.OuterClass.InnerClass", constants.ClassIsFinal, false)
	AssertMods(t, gc, "com.example.base.OuterClass.InnerClass", "public")
	AssertDef(t, gc, "com.example.base.OuterClass.StaticNestedClass", model.Class)
	AssertMores(t, gc, "com.example.base.OuterClass.StaticNestedClass", constants.ClassIsStatic, true)
	AssertMores(t, gc, "com.example.base.OuterClass.StaticNestedClass", constants.ClassIsAbstract, false)
	AssertMods(t, gc, "com.example.base.OuterClass.StaticNestedClass", "public", "static")
	// 内部类 / 嵌套类方法（void 返回 + 复杂度 + 修饰符）
	AssertDef(t, gc, "com.example.base.OuterClass.InnerClass.display()", model.Method)
	AssertMods(t, gc, "com.example.base.OuterClass.InnerClass.display()", "public")
	AssertMores(t, gc, "com.example.base.OuterClass.InnerClass.display()", constants.MethodReturnType, "void")
	AssertMores(t, gc, "com.example.base.OuterClass.InnerClass.display()", constants.MethodComplexity, 1)
	AssertDef(t, gc, "com.example.base.OuterClass.StaticNestedClass.run()", model.Method)
	AssertMods(t, gc, "com.example.base.OuterClass.StaticNestedClass.run()", "public", "static")
	AssertMores(t, gc, "com.example.base.OuterClass.StaticNestedClass.run()", constants.MethodReturnType, "void")
	AssertMores(t, gc, "com.example.base.OuterClass.StaticNestedClass.run()", constants.MethodComplexity, 1)

	// --- 方法内局部类 ---
	AssertDef(t, gc, "com.example.base.OuterClass.scopeTest().LocalClass", model.Class)
	AssertMores(t, gc, "com.example.base.OuterClass.scopeTest().LocalClass", constants.ClassIsStatic, false)
	AssertMores(t, gc, "com.example.base.OuterClass.scopeTest().LocalClass", constants.ClassIsAbstract, false)
	AssertDef(t, gc, "com.example.base.OuterClass.scopeTest().LocalClass.action()", model.Method)
	AssertMores(t, gc, "com.example.base.OuterClass.scopeTest().LocalClass.action()", constants.MethodReturnType, "void")
	AssertMores(t, gc, "com.example.base.OuterClass.scopeTest().LocalClass.action()", constants.MethodComplexity, 1)

	// --- 嵌套结构各类的隐式构造器合成 ---
	assertImplicitCtor(t, gc, "com.example.base.OuterClass.OuterClass()")
	assertImplicitCtor(t, gc, "com.example.base.OuterClass.InnerClass.InnerClass()")
	assertImplicitCtor(t, gc, "com.example.base.OuterClass.StaticNestedClass.StaticNestedClass()")
	assertImplicitCtor(t, gc, "com.example.base.OuterClass.scopeTest().LocalClass.LocalClass()")

	// --- CallbackManager：局部类 + 匿名类 ---
	AssertDef(t, gc, "com.example.base.CallbackManager.register().LocalValidator", model.Class)
	AssertDef(t, gc, "com.example.base.CallbackManager.register().LocalValidator.isValid()", model.Method)
	AssertMores(t, gc, "com.example.base.CallbackManager.register().LocalValidator.isValid()", constants.MethodReturnType, "boolean")
	AssertMores(t, gc, "com.example.base.CallbackManager.register().LocalValidator.isValid()", constants.MethodComplexity, 1)
	// 局部类隐式构造器
	assertImplicitCtor(t, gc, "com.example.base.CallbackManager.register().LocalValidator.LocalValidator()")

	// 匿名类宿主方法（复杂度/返回类型）
	AssertDef(t, gc, "com.example.base.CallbackManager.register()", model.Method)
	AssertMores(t, gc, "com.example.base.CallbackManager.register()", constants.MethodComplexity, 1)

	AssertDef(t, gc, "com.example.base.CallbackManager.register().anonymousClass$1", model.AnonymousClass)
	AssertMores(t, gc, "com.example.base.CallbackManager.register().anonymousClass$1", constants.AnonymousClassType, "Runnable")
	AssertDef(t, gc, "com.example.base.CallbackManager.register().anonymousClass$1.run()", model.Method)
	// 匿名类内方法 override 注解 + 元数据
	AssertAnnos(t, gc, "com.example.base.CallbackManager.register().anonymousClass$1.run()", "@Override")
	AssertMores(t, gc, "com.example.base.CallbackManager.register().anonymousClass$1.run()", constants.MethodReturnType, "void")
	AssertMores(t, gc, "com.example.base.CallbackManager.register().anonymousClass$1.run()", constants.MethodComplexity, 1)
	AssertMods(t, gc, "com.example.base.CallbackManager.register().anonymousClass$1.run()", "public")
}

// assertImplicitCtor 断言某构造器 QN 存在且为隐式合成构造器。
func assertImplicitCtor(t *testing.T, gc *core.GlobalContext, ctorQN string) {
	t.Helper()
	AssertDef(t, gc, ctorQN, model.Method)
	AssertMores(t, gc, ctorQN, constants.MethodIsConstructor, true)
	AssertMores(t, gc, ctorQN, constants.MethodIsImplicit, true)
}

// =============================================================================
// S6 原生语法糖：Record / 默认构造器 / Lambda / 方法引用 / try-with-resources（collector/sugar）
// =============================================================================

func TestCollectorS6SyntacticSugar(t *testing.T) {
	gc := RunCollection(t, "x/java/testdata", ResolvePaths(t, "collector/sugar"))

	// --- Record：组件字段 + 隐式访问器 + 隐式构造器 ---
	AssertDef(t, gc, "com.example.sugar.User", model.Class)
	AssertSig(t, gc, "com.example.sugar.User", "record")

	// 组件字段（is_record_component=true, is_final=true；FieldRawType 与 VariableRawType 双键齐备）
	AssertDef(t, gc, "com.example.sugar.User.id", model.Field)
	AssertMores(t, gc, "com.example.sugar.User.id", constants.FieldIsRecordComponent, true)
	AssertMores(t, gc, "com.example.sugar.User.id", constants.FieldIsFinal, true)
	AssertMores(t, gc, "com.example.sugar.User.id", constants.FieldRawType, "Long")
	AssertMores(t, gc, "com.example.sugar.User.id", constants.VariableRawType, "Long")
	// 组件字段保留 variable 语义（is_param=true，来源为 record 形参）
	AssertMores(t, gc, "com.example.sugar.User.id", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.sugar.User.name", constants.FieldIsRecordComponent, true)
	AssertMores(t, gc, "com.example.sugar.User.name", constants.FieldRawType, "String")
	AssertMores(t, gc, "com.example.sugar.User.name", constants.VariableRawType, "String")
	AssertMores(t, gc, "com.example.sugar.User.name", constants.VariableIsParam, true)

	// 显式定义的 name() 方法（普通方法，无隐式标记）
	AssertDef(t, gc, "com.example.sugar.User.name()", model.Method)
	AssertMores(t, gc, "com.example.sugar.User.name()", constants.MethodIsConstructor, false)

	// 隐式访问器 id() 与隐式构造器 User(Long,String)（IsFormSugar，元数据与 enricher 产出对齐）
	AssertDef(t, gc, "com.example.sugar.User.id()", model.Method)
	AssertMores(t, gc, "com.example.sugar.User.id()", constants.MethodIsImplicit, true)
	AssertMores(t, gc, "com.example.sugar.User.id()", constants.MethodIsConstructor, false)
	AssertMores(t, gc, "com.example.sugar.User.id()", constants.MethodReturnType, "Long")
	AssertDef(t, gc, "com.example.sugar.User.User(Long,String)", model.Method)
	AssertSig(t, gc, "com.example.sugar.User.User(Long,String)", "public User(Long id, String name)")
	AssertMores(t, gc, "com.example.sugar.User.User(Long,String)", constants.MethodIsImplicit, true)
	AssertMores(t, gc, "com.example.sugar.User.User(Long,String)", constants.MethodIsConstructor, true)
	AssertMores(t, gc, "com.example.sugar.User.User(Long,String)", constants.MethodParameters, []string{"Long id", "String name"})

	// Record 隐式 Object 方法：equals / hashCode / toString（IsFormSugar，含完整方法元数据）
	AssertDef(t, gc, "com.example.sugar.User.equals(Object)", model.Method)
	AssertMores(t, gc, "com.example.sugar.User.equals(Object)", constants.MethodIsImplicit, true)
	AssertMores(t, gc, "com.example.sugar.User.equals(Object)", constants.MethodIsConstructor, false)
	AssertMores(t, gc, "com.example.sugar.User.equals(Object)", constants.MethodReturnType, "boolean")
	AssertMores(t, gc, "com.example.sugar.User.equals(Object)", constants.MethodParameters, []string{"Object o"})
	AssertDef(t, gc, "com.example.sugar.User.hashCode()", model.Method)
	AssertMores(t, gc, "com.example.sugar.User.hashCode()", constants.MethodIsImplicit, true)
	AssertMores(t, gc, "com.example.sugar.User.hashCode()", constants.MethodReturnType, "int")
	AssertDef(t, gc, "com.example.sugar.User.toString()", model.Method)
	AssertMores(t, gc, "com.example.sugar.User.toString()", constants.MethodIsImplicit, true)
	AssertMores(t, gc, "com.example.sugar.User.toString()", constants.MethodReturnType, "String")

	// --- 默认构造器（无显式构造器时合成）---
	AssertDef(t, gc, "com.example.sugar.DefaultConstructor", model.Class)
	AssertDef(t, gc, "com.example.sugar.DefaultConstructor.DefaultConstructor()", model.Method)
	AssertMores(t, gc, "com.example.sugar.DefaultConstructor.DefaultConstructor()", constants.MethodIsImplicit, true)
	AssertMores(t, gc, "com.example.sugar.DefaultConstructor.DefaultConstructor()", constants.MethodIsConstructor, true)
	AssertSig(t, gc, "com.example.sugar.DefaultConstructor.DefaultConstructor()", "public DefaultConstructor()")

	// --- 枚举隐式方法（无参枚举 Color）---
	AssertDef(t, gc, "com.example.sugar.Color", model.Enum)
	// 枚举常量（签名=常量名）
	AssertDef(t, gc, "com.example.sugar.Color.RED", model.EnumConstant)
	AssertDef(t, gc, "com.example.sugar.Color.BLUE", model.EnumConstant)
	AssertSig(t, gc, "com.example.sugar.Color.RED", "RED")
	AssertDef(t, gc, "com.example.sugar.Color.values()", model.Method)
	AssertMores(t, gc, "com.example.sugar.Color.values()", constants.MethodIsImplicit, true)
	AssertMores(t, gc, "com.example.sugar.Color.values()", constants.MethodIsConstructor, false)
	AssertMores(t, gc, "com.example.sugar.Color.values()", constants.MethodReturnType, "Color[]")
	AssertSig(t, gc, "com.example.sugar.Color.values()", "public static Color[] values()")
	AssertDef(t, gc, "com.example.sugar.Color.valueOf(String)", model.Method)
	AssertMores(t, gc, "com.example.sugar.Color.valueOf(String)", constants.MethodIsImplicit, true)
	AssertMores(t, gc, "com.example.sugar.Color.valueOf(String)", constants.MethodIsConstructor, false)
	AssertMores(t, gc, "com.example.sugar.Color.valueOf(String)", constants.MethodReturnType, "Color")
	AssertMores(t, gc, "com.example.sugar.Color.valueOf(String)", constants.MethodParameters, []string{"String name"})

	// --- Lambda（表达式 lambda + 块 lambda + 推断/显式类型参数）---
	AssertDef(t, gc, "com.example.sugar.LambdaTest.testLambda()", model.Method)
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda()", constants.MethodReturnType, "void")
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda()", constants.MethodComplexity, 1)
	// 宿主变量类型（lambda 载体局部变量）
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().adder", constants.VariableRawType, "BiFunction<Integer, Integer, Integer>")
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().printer", constants.VariableRawType, "Consumer<String>")
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().multiplier", constants.VariableRawType, "BinaryOperator<Long>")

	AssertDef(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$1", model.Lambda)
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$1", constants.LambdaParameters, "(a, b)")
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$1", constants.LambdaBodyIsBlock, false)
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$1", constants.MethodComplexity, 1)
	// 推断类型参数（无显式类型）：is_param=false + raw_type=inferred
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$1.a", constants.VariableRawType, "inferred")
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$1.a", constants.VariableIsParam, false)
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$1.b", constants.VariableRawType, "inferred")
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$1.b", constants.VariableIsParam, false)
	// 单参数括号省略 + 块 lambda 体内变量
	AssertDef(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$2", model.Lambda)
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$2", constants.LambdaParameters, "s")
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$2", constants.LambdaBodyIsBlock, true)
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$2", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$2.s", constants.VariableRawType, "inferred")
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$2.s", constants.VariableIsParam, false)
	AssertDef(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$2.prefix", model.Variable)
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$2.prefix", constants.VariableRawType, "String")
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$2.prefix", constants.VariableIsParam, false)
	// 显式类型参数：is_param=true + raw_type 保留类型（与推断参数形成对比）
	AssertDef(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$3", model.Lambda)
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$3", constants.LambdaParameters, "(Long x, Long y)")
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$3", constants.LambdaBodyIsBlock, false)
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$3", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$3.x", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$3.x", constants.VariableRawType, "Long")
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$3.y", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.sugar.LambdaTest.testLambda().lambda$3.y", constants.VariableRawType, "Long")

	// --- 方法引用（6 种形态：静态/绑定/任意/构造器/数组/泛型）---
	mref := "com.example.sugar.MethodRefTest.testAllMethodReferences()"
	AssertDef(t, gc, mref+".method_ref$1", model.MethodRef)
	AssertMores(t, gc, mref+".method_ref$1", constants.MethodRefReceiver, "Integer")
	AssertMores(t, gc, mref+".method_ref$1", constants.MethodRefTarget, "parseInt")
	AssertSig(t, gc, mref+".method_ref$1", "Integer::parseInt")
	AssertMores(t, gc, mref+".method_ref$2", constants.MethodRefReceiver, "System.out")
	AssertMores(t, gc, mref+".method_ref$2", constants.MethodRefTarget, "println")
	AssertSig(t, gc, mref+".method_ref$2", "System.out::println")
	AssertMores(t, gc, mref+".method_ref$3", constants.MethodRefReceiver, "String")
	AssertMores(t, gc, mref+".method_ref$3", constants.MethodRefTarget, "toLowerCase")
	AssertSig(t, gc, mref+".method_ref$3", "String::toLowerCase")
	AssertMores(t, gc, mref+".method_ref$4", constants.MethodRefReceiver, "ArrayList")
	AssertMores(t, gc, mref+".method_ref$4", constants.MethodRefTarget, "new")
	AssertSig(t, gc, mref+".method_ref$4", "ArrayList::new")
	AssertMores(t, gc, mref+".method_ref$5", constants.MethodRefReceiver, "int[]")
	AssertMores(t, gc, mref+".method_ref$5", constants.MethodRefTarget, "new")
	AssertSig(t, gc, mref+".method_ref$5", "int[]::new")
	AssertMores(t, gc, mref+".method_ref$6", constants.MethodRefReceiver, "this")
	AssertMores(t, gc, mref+".method_ref$6", constants.MethodRefTarget, "genericMethod")
	AssertMores(t, gc, mref+".method_ref$6", constants.MethodRefTypeArgs, "<String>")
	AssertSig(t, gc, mref+".method_ref$6", "this::<String>genericMethod")
	// 泛型方法定义与宿主方法
	AssertDef(t, gc, "com.example.sugar.MethodRefTest.genericMethod(T)", model.Method)
	AssertMores(t, gc, "com.example.sugar.MethodRefTest.genericMethod(T)", constants.MethodReturnType, "void")
	AssertMores(t, gc, "com.example.sugar.MethodRefTest.genericMethod(T)", constants.MethodComplexity, 1)
	AssertMores(t, gc, "com.example.sugar.MethodRefTest.genericMethod(T).t", constants.VariableIsParam, true)
	AssertMores(t, gc, "com.example.sugar.MethodRefTest.genericMethod(T).t", constants.VariableRawType, "T")
	AssertDef(t, gc, mref, model.Method)
	AssertMores(t, gc, mref, constants.MethodComplexity, 1)

	// --- try-with-resources（资源变量重定位到 block$1/block$2，is_param=false）---
	AssertDef(t, gc, "com.example.sugar.TryWithResourcesTest.test().block$1.input", model.Variable)
	AssertMores(t, gc, "com.example.sugar.TryWithResourcesTest.test().block$1.input", constants.VariableRawType, "InputStream")
	AssertMores(t, gc, "com.example.sugar.TryWithResourcesTest.test().block$1.input", constants.VariableIsParam, false)
	AssertMores(t, gc, "com.example.sugar.TryWithResourcesTest.test().block$1", constants.MethodComplexity, 1)
	AssertDef(t, gc, "com.example.sugar.TryWithResourcesTest.test().block$2.out", model.Variable)
	AssertMores(t, gc, "com.example.sugar.TryWithResourcesTest.test().block$2.out", constants.VariableRawType, "OutputStream")
	AssertMores(t, gc, "com.example.sugar.TryWithResourcesTest.test().block$2.out", constants.VariableIsParam, false)
	AssertDef(t, gc, "com.example.sugar.TryWithResourcesTest.test().block$2.in", model.Variable)
	AssertMores(t, gc, "com.example.sugar.TryWithResourcesTest.test().block$2.in", constants.VariableRawType, "InputStream")
	AssertMores(t, gc, "com.example.sugar.TryWithResourcesTest.test().block$2.in", constants.VariableIsParam, false)
	AssertMores(t, gc, "com.example.sugar.TryWithResourcesTest.test().block$2", constants.MethodComplexity, 1)
	// 宿主方法：throws + 复杂度 + Signature
	AssertDef(t, gc, "com.example.sugar.TryWithResourcesTest.test()", model.Method)
	AssertMores(t, gc, "com.example.sugar.TryWithResourcesTest.test()", constants.MethodThrowsTypes, []string{"IOException"})
	AssertMores(t, gc, "com.example.sugar.TryWithResourcesTest.test()", constants.MethodComplexity, 1)
	AssertSig(t, gc, "com.example.sugar.TryWithResourcesTest.test()", "public void test() throws IOException")
}

// =============================================================================
// S9 泛型：泛型接口 / 通配符 / 类型参数方法（collector/generic）
// =============================================================================

func TestCollectorS9Generic(t *testing.T) {
	gc := RunCollection(t, "x/java/testdata", ResolvePaths(t, "collector/generic"))

	// --- 泛型接口（T extends Serializable & Cloneable）---
	AssertDef(t, gc, "com.example.base.test.GenericTest", model.Interface)

	// --- 通配符方法（extends / super 边界）---
	AssertDef(t, gc, "com.example.base.test.GenericTest.findAllByCriteria(List)", model.Method)
	AssertMores(t, gc, "com.example.base.test.GenericTest.findAllByCriteria(List)", constants.MethodReturnType, "List<? extends T>")
	AssertMores(t, gc, "com.example.base.test.GenericTest.findAllByCriteria(List)", constants.MethodParameters, []string{"List<? super T> criteria"})
	AssertMores(t, gc, "com.example.base.test.GenericTest.findAllByCriteria(List).criteria", constants.VariableRawType, "List<? super T>")

	// --- 泛型方法（<E extends Exception>，throws E）---
	AssertDef(t, gc, "com.example.base.test.GenericTest.executeOrThrow(E)", model.Method)
	AssertMores(t, gc, "com.example.base.test.GenericTest.executeOrThrow(E)", constants.MethodParameters, []string{"E exception"})
	AssertMores(t, gc, "com.example.base.test.GenericTest.executeOrThrow(E)", constants.MethodThrowsTypes, []string{"E"})
	AssertSig(t, gc, "com.example.base.test.GenericTest.executeOrThrow(E)", "<E extends Exception> void executeOrThrow(E exception) throws E")

	// --- 接口多界类型参数（ClassTypeParameters 此前未断言）---
	AssertMores(t, gc, "com.example.base.test.GenericTest", constants.ClassTypeParameters, "<T extends Serializable & Cloneable>")

	// --- 泛型边界（GenericEdge.java）：多类型参数/泛型字段/泛型方法/泛型构造器/嵌套通配符 ---
	ge := "com.example.base.test.GenericEdge"
	AssertDef(t, gc, ge, model.Class)
	AssertMores(t, gc, ge, constants.ClassTypeParameters, "<K extends Number, V extends Comparable<V> & Cloneable>")

	// 泛型字段：嵌套泛型 / 无界通配符 / 上界通配符 / 下界通配符
	AssertMores(t, gc, ge+".names", constants.FieldRawType, "List<String>")
	AssertMores(t, gc, ge+".index", constants.FieldRawType, "Map<String, List<Integer>>")
	AssertMores(t, gc, ge+".rawWildcard", constants.FieldRawType, "List<?>")
	AssertMores(t, gc, ge+".upperWildcard", constants.FieldRawType, "List<? extends Number>")
	AssertMores(t, gc, ge+".lowerWildcard", constants.FieldRawType, "List<? super Integer>")

	// 泛型方法：返回类型参数 / 泛型 identity / 下界通配符参数
	AssertMores(t, gc, ge+".firstKey()", constants.MethodReturnType, "K")
	AssertMores(t, gc, ge+".identity(T)", constants.MethodReturnType, "T")
	AssertMores(t, gc, ge+".identity(T)", constants.MethodParameters, []string{"T value"})
	AssertMores(t, gc, ge+".consume(List)", constants.MethodParameters, []string{"List<? super V> sink"})

	// 泛型构造器
	AssertDef(t, gc, ge+".GenericEdge(E,List)", model.Method)
	AssertMores(t, gc, ge+".GenericEdge(E,List)", constants.MethodIsConstructor, true)
	AssertMores(t, gc, ge+".GenericEdge(E,List)", constants.MethodParameters, []string{"E seed", "List<K> keys"})

	// 泛型方法多界（Signature 含 <E extends Number & Comparable<E>>，Mores 存 MethodTypeParameters）
	AssertSig(t, gc, ge+".compareWith(E)", "<E extends Number & Comparable<E>> int compareWith(E other)")
	AssertMores(t, gc, ge+".compareWith(E)", constants.MethodTypeParameters, "<E extends Number & Comparable<E>>")
	AssertMores(t, gc, ge+".identity(T)", constants.MethodTypeParameters, "<T>")

	// 嵌套通配符：? extends List<? super K>
	AssertMores(t, gc, ge+".nestedWildcard(List)", constants.MethodParameters, []string{"List<? extends List<? super K>> deep"})
	AssertMores(t, gc, ge+".nestedWildcard(List).deep", constants.VariableRawType, "List<? extends List<? super K>>")
}

// =============================================================================
// S7 Lombok 全系语法糖：11 个样例文件逐一断言生成的方法/字段（collector/sugar/lombok）
// =============================================================================

func TestCollectorS7Lombok(t *testing.T) {
	gc := RunCollection(t, "x/java/testdata", ResolvePaths(t, "collector/sugar/lombok"))

	// --- @Slf4j：生成静态日志字段（FieldRawType 为字段标准键，供类型解析）---
	t.Run("Slf4j", func(t *testing.T) {
		for _, qn := range []string{
			"com.example.lombok.Slf4jExample.log",
			"com.example.lombok.MultipleAnnotationsExample.log",
		} {
			AssertDef(t, gc, qn, model.Field)
			AssertMores(t, gc, qn, constants.FieldIsStatic, true)
			AssertMores(t, gc, qn, constants.FieldIsFinal, true)
			AssertMores(t, gc, qn, constants.LombokAnnotationType, "Slf4j")
			AssertMores(t, gc, qn, constants.FieldRawType, "org.slf4j.Logger")
		}
	})

	// --- @Getter：getX() / isX() ---
	t.Run("Getter", func(t *testing.T) {
		cases := []struct {
			qn    string
			sig   string
			rtype string
		}{
			{"com.example.lombok.GetterOnly.getName()", "public String getName()", "String"},
			{"com.example.lombok.GetterOnly.isActive()", "public boolean isActive()", "boolean"},
			{"com.example.lombok.DataComprehensive.getAge()", "public int getAge()", "int"},
			{"com.example.lombok.ValueExample.getName()", "public String getName()", "String"},
			{"com.example.lombok.EqualsAndHashCodeExample.getVersion()", "public int getVersion()", "int"},
			{"com.example.lombok.MultipleAnnotationsExample.getValue()", "public int getValue()", "int"},
			{"com.example.lombok.EdgeCases.getPrimitiveInt()", "public int getPrimitiveInt()", "int"},
			// final 字段有 getter（GetterOnly.age 是 final）
			{"com.example.lombok.GetterOnly.getAge()", "public int getAge()", "int"},
			// Boolean 包装类型走 is 前缀（返回类型保持包装类型 Boolean）
			{"com.example.lombok.EdgeCases.isWrapperBoolean()", "public Boolean isWrapperBoolean()", "Boolean"},
			// 数组字段 getter
			{"com.example.lombok.EdgeCases.getIntArray()", "public int[] getIntArray()", "int[]"},
		}
		for _, c := range cases {
			AssertDef(t, gc, c.qn, model.Method)
			AssertMores(t, gc, c.qn, constants.MethodIsLombokGenerated, true)
			AssertMores(t, gc, c.qn, constants.LombokAnnotationType, "Getter")
			AssertMores(t, gc, c.qn, constants.MethodReturnType, c.rtype)
			AssertMores(t, gc, c.qn, constants.MethodIsConstructor, false)
		}
	})

	// --- @Setter：setX(T) ---
	t.Run("Setter", func(t *testing.T) {
		cases := []struct {
			qn     string
			params []string
		}{
			{"com.example.lombok.SetterOnly.setName(String)", []string{"String name"}},
			{"com.example.lombok.SetterOnly.setActive(boolean)", []string{"boolean active"}},
			{"com.example.lombok.DataComprehensive.setEmail(String)", []string{"String email"}},
			{"com.example.lombok.EqualsAndHashCodeExample.setVersion(int)", []string{"int version"}},
			{"com.example.lombok.MultipleAnnotationsExample.setId(String)", []string{"String id"}},
			{"com.example.lombok.EdgeCases.setWrapperInt(Integer)", []string{"Integer wrapperInt"}},
			// 数组字段 setter
			{"com.example.lombok.EdgeCases.setIntArray(int[])", []string{"int[] intArray"}},
			{"com.example.lombok.EdgeCases.setStringArray(String[])", []string{"String[] stringArray"}},
		}
		for _, c := range cases {
			AssertDef(t, gc, c.qn, model.Method)
			AssertMores(t, gc, c.qn, constants.MethodIsLombokGenerated, true)
			AssertMores(t, gc, c.qn, constants.LombokAnnotationType, "Setter")
			AssertMores(t, gc, c.qn, constants.MethodParameters, c.params)
			AssertMores(t, gc, c.qn, constants.MethodIsConstructor, false)
		}
	})

	// --- Getter/Setter 边界：static 字段不生成 getter/setter ---
	t.Run("StaticNegatives", func(t *testing.T) {
		// EdgeCases.staticField 是 static，@Data 不为其生成访问器
		AssertNoDef(t, gc, "com.example.lombok.EdgeCases.getStaticField()")
		AssertNoDef(t, gc, "com.example.lombok.EdgeCases.setStaticField(String)")
		// 但 static 字段本身存在
		AssertDef(t, gc, "com.example.lombok.EdgeCases.staticField", model.Field)
	})

	// --- @ToString：toString() ---
	t.Run("ToString", func(t *testing.T) {
		for _, qn := range []string{
			"com.example.lombok.ToStringExample.toString()",
			"com.example.lombok.ToStringDetailedExample.toString()",
			"com.example.lombok.DataComprehensive.toString()",
			"com.example.lombok.ValueExample.toString()",
			"com.example.lombok.EdgeCases.toString()",
			"com.example.lombok.MultipleAnnotationsExample.toString()",
		} {
			AssertDef(t, gc, qn, model.Method)
			AssertMores(t, gc, qn, constants.MethodIsLombokGenerated, true)
			AssertMores(t, gc, qn, constants.LombokAnnotationType, "ToString")
			AssertMores(t, gc, qn, constants.MethodReturnType, "String")
			AssertMores(t, gc, qn, constants.MethodIsConstructor, false)
		}
	})

	// --- @EqualsAndHashCode：equals / hashCode / canEqual ---
	t.Run("EqualsAndHashCode", func(t *testing.T) {
		for _, qn := range []string{
			"com.example.lombok.EqualsAndHashCodeExample.equals(Object)",
			"com.example.lombok.EqualsAndHashCodeExample.hashCode()",
			"com.example.lombok.EqualsAndHashCodeExample.canEqual(Object)",
			"com.example.lombok.DataComprehensive.equals(Object)",
			"com.example.lombok.DataComprehensive.canEqual(Object)",
			"com.example.lombok.ValueExample.hashCode()",
			"com.example.lombok.ValueExample.canEqual(Object)",
			"com.example.lombok.EdgeCases.equals(Object)",
			"com.example.lombok.MultipleAnnotationsExample.hashCode()",
		} {
			AssertDef(t, gc, qn, model.Method)
			AssertMores(t, gc, qn, constants.MethodIsLombokGenerated, true)
			AssertMores(t, gc, qn, constants.LombokAnnotationType, "EqualsAndHashCode")
			AssertMores(t, gc, qn, constants.MethodIsConstructor, false)
		}
	})

	// --- @AllArgsConstructor：全参构造器 ---
	t.Run("AllArgsConstructor", func(t *testing.T) {
		cases := []struct {
			qn     string
			params []string
		}{
			{"com.example.lombok.ConstructorComprehensive.ConstructorComprehensive(String,int)", []string{"String name", "int age"}},
			{"com.example.lombok.BuilderComprehensive.BuilderComprehensive(String,int,boolean,String,String)", []string{"String host", "int port", "boolean secure", "String username", "String password"}},
			{"com.example.lombok.ValueExample.ValueExample(String,int,boolean)", []string{"String name", "int age", "boolean active"}},
		}
		for _, c := range cases {
			AssertDef(t, gc, c.qn, model.Method)
			AssertMores(t, gc, c.qn, constants.MethodIsLombokGenerated, true)
			AssertMores(t, gc, c.qn, constants.MethodIsConstructor, true)
			AssertMores(t, gc, c.qn, constants.LombokAnnotationType, "AllArgsConstructor")
			AssertMores(t, gc, c.qn, constants.MethodParameters, c.params)
		}
	})

	// --- @NoArgsConstructor：无参构造器（Lombok 注解生成，区别于默认构造器合成）---
	t.Run("NoArgsConstructor", func(t *testing.T) {
		cases := []struct {
			qn     string
			sig    string
			lombok string
		}{
			{"com.example.lombok.ConstructorComprehensive.ConstructorComprehensive()", "public ConstructorComprehensive()", "NoArgsConstructor"},
			{"com.example.lombok.BuilderComprehensive.BuilderComprehensive()", "public BuilderComprehensive()", "NoArgsConstructor"},
			// @Data 内部委托 lombokNoArgsConstructor，注解类型标记为 NoArgsConstructor
			{"com.example.lombok.DataComprehensive.DataComprehensive()", "public DataComprehensive()", "NoArgsConstructor"},
		}
		for _, c := range cases {
			AssertDef(t, gc, c.qn, model.Method)
			AssertMores(t, gc, c.qn, constants.MethodIsLombokGenerated, true)
			AssertMores(t, gc, c.qn, constants.MethodIsConstructor, true)
			AssertMores(t, gc, c.qn, constants.LombokAnnotationType, c.lombok)
			// 无参构造器不写 MethodParameters 键（与 MethodEnricher 对齐）
			if _, ok := FindDef(t, gc, c.qn).GetMores()[constants.MethodParameters]; ok {
				t.Errorf("%s should not have Mores[%s]", c.qn, constants.MethodParameters)
			}
		}
	})

	// --- 构造器负向：@Value/@RequiredArgsConstructor/@Builder 类不生成无参构造器 ---
	t.Run("ConstructorNegatives", func(t *testing.T) {
		for _, qn := range []string{
			"com.example.lombok.ValueExample.ValueExample()",                                                // @Value 只生成 AllArgsConstructor
			"com.example.lombok.ConstructorComprehensive.RequiredArgsConstructor.RequiredArgsConstructor()", // @RequiredArgsConstructor 只生成必选构造器
			"com.example.lombok.ConstructorComprehensive.RequiredArgsConstructor.InnerClass.InnerClass()",   // 同上（嵌套类）
			"com.example.lombok.BuilderComprehensive.Config.Config()",                                       // @Builder 无 @NoArgsConstructor 时不生成无参
		} {
			AssertNoDef(t, gc, qn)
		}
		// 反向守卫：ValueExample 有全参构造器、无 setter（@Value 不可变）
		AssertDef(t, gc, "com.example.lombok.ValueExample.ValueExample(String,int,boolean)", model.Method)
		AssertNoDef(t, gc, "com.example.lombok.ValueExample.setName(String)")
		AssertNoDef(t, gc, "com.example.lombok.ValueExample.setActive(boolean)")
	})

	// --- @RequiredArgsConstructor：必选字段构造器 ---
	t.Run("RequiredArgsConstructor", func(t *testing.T) {
		cases := []struct {
			qn     string
			params []string
		}{
			// preinitialized 带初始化器（= "fixed"），不进入构造器参数
			{"com.example.lombok.ConstructorComprehensive.RequiredArgsConstructor.RequiredArgsConstructor(String)", []string{"String requiredField"}},
			{"com.example.lombok.ConstructorComprehensive.RequiredArgsConstructor.InnerClass.InnerClass(int)", []string{"int id"}},
		}
		for _, c := range cases {
			AssertDef(t, gc, c.qn, model.Method)
			AssertMores(t, gc, c.qn, constants.MethodIsLombokGenerated, true)
			AssertMores(t, gc, c.qn, constants.MethodIsConstructor, true)
			AssertMores(t, gc, c.qn, constants.LombokAnnotationType, "RequiredArgsConstructor")
			AssertMores(t, gc, c.qn, constants.MethodParameters, c.params)
		}
		// 带初始化器的 final 字段不产生构造器参数：
		// 若解析器误把 preinitialized 收进构造器，会产出 2 参构造器 (String,String)
		AssertNoDef(t, gc, "com.example.lombok.ConstructorComprehensive.RequiredArgsConstructor.RequiredArgsConstructor(String,String)")
	})

	// --- @Builder：builder() + Builder 内部类 + 链式方法 + build() ---
	t.Run("Builder", func(t *testing.T) {
		AssertDef(t, gc, "com.example.lombok.BuilderComprehensive.builder()", model.Method)
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.builder()", constants.MethodIsLombokGenerated, true)
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.builder()", constants.LombokAnnotationType, "Builder")
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.builder()", constants.MethodIsConstructor, false)

		AssertDef(t, gc, "com.example.lombok.BuilderComprehensive.Builder", model.Class)
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.Builder", constants.LombokAnnotationType, "Builder")
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.Builder", constants.MethodIsLombokGenerated, true)
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.Builder", constants.ClassIsStatic, true)

		AssertDef(t, gc, "com.example.lombok.BuilderComprehensive.Builder.host(String)", model.Method)
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.Builder.host(String)", constants.MethodIsLombokGenerated, true)
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.Builder.host(String)", constants.MethodReturnType, "Builder")
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.Builder.host(String)", constants.MethodIsConstructor, false)
		AssertDef(t, gc, "com.example.lombok.BuilderComprehensive.Builder.build()", model.Method)
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.Builder.build()", constants.MethodReturnType, "BuilderComprehensive")
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.Builder.build()", constants.MethodIsConstructor, false)

		// 嵌套 Config 类的 Builder
		AssertDef(t, gc, "com.example.lombok.BuilderComprehensive.Config.builder()", model.Method)
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.Config.builder()", constants.LombokAnnotationType, "Builder")
		AssertDef(t, gc, "com.example.lombok.BuilderComprehensive.Config.Builder.database(String)", model.Method)
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.Config.Builder.database(String)", constants.MethodReturnType, "Builder")
		AssertDef(t, gc, "com.example.lombok.BuilderComprehensive.Config.Builder.build()", model.Method)
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.Config.Builder.build()", constants.MethodReturnType, "Config")

		// Builder 专用私有构造器
		AssertDef(t, gc, "com.example.lombok.BuilderComprehensive.BuilderComprehensive(Builder)", model.Method)
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.BuilderComprehensive(Builder)", constants.MethodIsLombokGenerated, true)
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.BuilderComprehensive(Builder)", constants.MethodIsConstructor, true)
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.BuilderComprehensive(Builder)", constants.LombokAnnotationType, "Builder")
		AssertMores(t, gc, "com.example.lombok.BuilderComprehensive.BuilderComprehensive(Builder)", constants.MethodParameters, []string{"Builder builder"})
	})

	// --- 源码元素保持 IsFormSource，生成元素为 IsFormSugar ---
	t.Run("FormFlags", func(t *testing.T) {
		src := FindDef(t, gc, "com.example.lombok.BuilderComprehensive.host")
		if !src.IsFormSource {
			t.Error("host field should be IsFormSource")
		}
		sugar := FindDef(t, gc, "com.example.lombok.BuilderComprehensive.builder()")
		if !sugar.IsFormSugar {
			t.Error("builder() should be IsFormSugar")
		}
	})
}

// =============================================================================
// S8 度量：LOC / 圈复杂度（collector/metrics）
// =============================================================================

func TestCollectorS8Metrics(t *testing.T) {
	gc := RunCollection(t, "x/java/testdata", ResolvePaths(t, "collector/metrics/loc", "collector/metrics/complexity"))

	// --- LOC 数值（loc_raw = 原始行数，loc = 去除注释/空行后）---
	// metrics/loc 四文件：Main 类共享 com.example 包，文件按 Name 区分
	assertFileLOC(t, gc, "SingleLine.java", 6, 10)
	assertFileLOC(t, gc, "PureCode.java", 6, 8)
	assertFileLOC(t, gc, "BlockComment.java", 7, 13)
	assertFileLOC(t, gc, "Complex.java", 6, 12)

	// --- 文件元素：包 com.example 下 Main 类存在 ---
	AssertDef(t, gc, "com.example.Main", model.Class)

	// --- 圈复杂度数值 ---
	c := "com.example.metrics.ComplexityTest"
	AssertDef(t, gc, c, model.Class)
	AssertMores(t, gc, c+".simple()", constants.MethodComplexity, 1)
	AssertMores(t, gc, c+".medium(int)", constants.MethodComplexity, 4)
	AssertMores(t, gc, c+".complexSwitch(int)", constants.MethodComplexity, 4)
	AssertMores(t, gc, c+".withTryCatch()", constants.MethodComplexity, 3)
	AssertMores(t, gc, c+".ternary(int)", constants.MethodComplexity, 2)
	AssertMores(t, gc, c+".arrowSwitch(int)", constants.MethodComplexity, 4)

	// 嵌套块复杂度（medium 内部 for 块 block$1.block$1 复杂度 1）
	AssertDef(t, gc, c+".medium(int).block$1.block$1", model.ScopeBlock)
	AssertMores(t, gc, c+".medium(int).block$1.block$1", constants.MethodComplexity, 1)
	// medium 外层 if 块（含 if 判定）复杂度 2
	AssertMores(t, gc, c+".medium(int).block$1", constants.MethodComplexity, 2)
	// withTryCatch 的 try 块（含 while）复杂度 2，catch 块复杂度 1
	AssertMores(t, gc, c+".withTryCatch().block$1", constants.MethodComplexity, 2)
	AssertMores(t, gc, c+".withTryCatch().block$2", constants.MethodComplexity, 1)

	// --- 边界构造（ComplexityEdge.java）：do/||/lambda/多 catch/嵌套三元 ---
	e := "com.example.metrics.ComplexityEdge"
	// do_statement：do 计入 +1，while 条件是 do_statement 子节点不单独计 => 2
	AssertMores(t, gc, e+".doLoop()", constants.MethodComplexity, 2)
	// || 与 && 各 +1 => 3
	AssertMores(t, gc, e+".logicOps(int,int)", constants.MethodComplexity, 3)
	// lambda 表达式 base=1，方法本身不含 lambda 复杂度增量
	AssertMores(t, gc, e+".lambdaSimple()", constants.MethodComplexity, 1)
	AssertDef(t, gc, e+".lambdaSimple().lambda$1", model.Lambda)
	AssertMores(t, gc, e+".lambdaSimple().lambda$1", constants.MethodComplexity, 1)
	// lambda 块内含 if+&&：lambda 复杂度 3，且方法复杂度递归计入 lambda 内判定 => 3
	AssertMores(t, gc, e+".lambdaWithLogic().lambda$1", constants.MethodComplexity, 3)
	AssertMores(t, gc, e+".lambdaWithLogic()", constants.MethodComplexity, 3)
	// 多 catch：base + 2 catch => 3
	AssertMores(t, gc, e+".multiCatch()", constants.MethodComplexity, 3)
	// 嵌套三元：base + 2 个 conditional_expression => 3
	AssertMores(t, gc, e+".nestedTernary(int)", constants.MethodComplexity, 3)
	// instanceof + if + || + 三元 => 4；if 块（含 || 与三元）=> 3
	AssertMores(t, gc, e+".complexReturn(Object)", constants.MethodComplexity, 4)
	AssertMores(t, gc, e+".complexReturn(Object).block$1", constants.MethodComplexity, 3)

	// --- ComplexityTest.java / ComplexityEdge.java 文件 LOC ---
	assertFileLOC(t, gc, "ComplexityTest.java", 43, 56)
	assertFileLOC(t, gc, "ComplexityEdge.java", 42, 60)
}

// assertFileLOC 断言指定文件名 FILE 元素的 LOC 数值。
func assertFileLOC(t *testing.T, gc *core.GlobalContext, name string, loc, locRaw int) {
	t.Helper()
	f := findFile(t, gc, name)
	AssertMores(t, gc, f.QualifiedName, constants.FileLOC, loc)
	AssertMores(t, gc, f.QualifiedName, constants.FileRawLOC, locRaw)
}

// =============================================================================
// S10 消歧盲区：同容器重名普通元素的 $N 条件序号（collector/dup）
// 验证 docs/qn-format.md §4.2：首个不带 $N，从第 2 个开始 $2、$3...
// =============================================================================

func TestCollectorS10DedupBlindspot(t *testing.T) {
	gc := RunCollection(t, "x/java/testdata", ResolvePaths(t, "collector/dup"))
	pkg := "com.example.base.test.DupNameTest"

	// --- 嵌套类 Inner（首个出现，不带 $N）---
	AssertDef(t, gc, pkg+".Inner", model.Class)
	AssertMores(t, gc, pkg+".Inner", constants.ClassIsStatic, false)

	// --- 同名字段（第 2 个出现）→ 消歧为 Inner$2 ---
	AssertDef(t, gc, pkg+".Inner$2", model.Field)
	AssertMores(t, gc, pkg+".Inner$2", constants.FieldRawType, "int")
	AssertMores(t, gc, pkg+".Inner$2", constants.FieldIsStatic, false)
	AssertSig(t, gc, pkg+".Inner$2", "int Inner")

	// --- 静态嵌套类 Config / 同名字段 Config$2 ---
	AssertDef(t, gc, pkg+".Config", model.Class)
	AssertMores(t, gc, pkg+".Config", constants.ClassIsStatic, true)
	AssertDef(t, gc, pkg+".Config$2", model.Field)
	AssertMores(t, gc, pkg+".Config$2", constants.FieldRawType, "String")
	AssertMores(t, gc, pkg+".Config$2", constants.FieldIsStatic, true)
	AssertSig(t, gc, pkg+".Config$2", "static String Config")

	// --- 不重名普通元素保持原名 ---
	AssertDef(t, gc, pkg+".count", model.Field)
	AssertMores(t, gc, pkg+".count", constants.FieldRawType, "int")

	// --- 方法内局部类 / 局部变量 ---
	AssertDef(t, gc, pkg+".test().Local", model.Class)
	AssertDef(t, gc, pkg+".test().ref", model.Variable)
	AssertMores(t, gc, pkg+".test().ref", constants.VariableRawType, "Local")

	// --- 不同 block 的同名变量独立计数，不触发 $N ---
	AssertDef(t, gc, pkg+".test().block$1.dup", model.Variable)
	AssertMores(t, gc, pkg+".test().block$1.dup", constants.VariableRawType, "int")
	AssertDef(t, gc, pkg+".test().block$2.dup", model.Variable)
	AssertMores(t, gc, pkg+".test().block$2.dup", constants.VariableRawType, "int")

	// --- 消歧后各嵌套类隐式构造器仍正常合成 ---
	assertImplicitCtor(t, gc, pkg+".Inner.Inner()")
	assertImplicitCtor(t, gc, pkg+".Config.Config()")
	assertImplicitCtor(t, gc, pkg+".test().Local.Local()")

	// --- 消歧后不应存在裸 Inner 字段 / 裸 Config 字段（首个不带 $N 的是类，字段直接进 $2）---
	AssertNoDef(t, gc, pkg+".Inner$1")
	AssertNoDef(t, gc, pkg+".Config$1")
}
