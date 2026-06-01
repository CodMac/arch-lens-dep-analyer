package collector

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestJavaCollector_AbstractBaseEntity(t *testing.T) {
	// 1. 获取测试文件路径
	filePath := test.GetTestFilePath(filepath.Join("collector", "base", "AbstractBaseEntity.java"))

	// 2. 解析源码
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 3. 运行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	// 断言 1: 包名验证
	expectedPackage := "com.example.base"
	if fCtx.PackageName != expectedPackage {
		t.Errorf("Expected PackageName %q, got %q", expectedPackage, fCtx.PackageName)
	}

	// 断言 2: Imports 数量及内容验证
	t.Run("Verify Imports", func(t *testing.T) {
		expectedImports := []string{"java.io.Serializable", "java.util.Date"}
		if len(fCtx.Imports) != len(expectedImports) {
			t.Errorf("Expected %d imports, got %d", len(expectedImports), len(fCtx.Imports))
		}

		for _, path := range expectedImports {
			parts := strings.Split(path, ".")
			alias := parts[len(parts)-1]
			if imps, ok := fCtx.Imports[alias]; !ok || imps[0].RawImportPath != path {
				t.Errorf("Missing or incorrect import for %s", path)
			}
		}
	})

	// 断言 3: 类定义、QN、Kind、Abstract 属性、签名验证
	t.Run("Verify AbstractBaseEntity Class", func(t *testing.T) {
		qn := "com.example.base.AbstractBaseEntity"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Definition not found for QN: %s", qn)
		}

		elem := defs[0].Element
		if elem.Kind != model.Class {
			t.Errorf("Expected Kind CLASS, got %s", elem.Kind)
		}

		if isAbs, ok := elem.Extra.Mores[java.ClassIsAbstract].(bool); !ok || !isAbs {
			t.Error("Expected java.class.is_abstract to be true")
		}

		// 验证签名 (注意：由于 JavaCollector 内部实现可能不同，这里匹配核心部分)
		expectedSign := "public abstract class AbstractBaseEntity<ID> implements Serializable"
		if expectedSign != elem.Signature {
			t.Errorf("Signature mismatch. Got: %q, Expected: %s", elem.Signature, expectedSign)
		}
	})

	// 断言 4: 字段 id 验证
	t.Run("Verify Field id", func(t *testing.T) {
		qn := "com.example.base.AbstractBaseEntity.id"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Field id not found")
		}

		elem := defs[0].Element
		if elem.Kind != model.Field {
			t.Errorf("Expected Field, got %s", elem.Kind)
		}

		if tpe := elem.Extra.Mores[java.FieldRawType]; tpe != "ID" {
			t.Errorf("Expected type ID, got %v", tpe)
		}

		if !contains(elem.Extra.Modifiers, "protected") {
			t.Error("Modifiers should contain 'protected'")
		}
	})

	// 断言 5: 字段 createdAt 验证
	t.Run("Verify Field createdAt", func(t *testing.T) {
		qn := "com.example.base.AbstractBaseEntity.createdAt"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Field createdAt not found")
		}

		elem := defs[0].Element
		if tpe := elem.Extra.Mores[java.FieldRawType]; tpe != "Date" {
			t.Errorf("Expected type Date, got %v", tpe)
		}

		if !contains(elem.Extra.Modifiers, "private") {
			t.Error("Modifiers should contain 'private'")
		}
	})

	// 断言 6 & 7: 方法 getId 和 setId 验证
	t.Run("Verify Methods", func(t *testing.T) {
		// getId()
		getIdQN := "com.example.base.AbstractBaseEntity.getId()"
		getDefs := test.FindDefinitionsByQN(fCtx, getIdQN)
		if len(getDefs) == 0 {
			t.Fatalf("Method getId() not found")
		}

		getElem := getDefs[0].Element
		if ret := getElem.Extra.Mores[java.MethodReturnType]; ret != "ID" {
			t.Errorf("getId expected return ID, got %v", ret)
		}

		// setId(ID id) - 验证 QN 括号内为类型
		setIdQN := "com.example.base.AbstractBaseEntity.setId(ID)"
		setDefs := test.FindDefinitionsByQN(fCtx, setIdQN)
		if len(setDefs) == 0 {
			t.Fatalf("Method setId(ID) not found")
		}

		setElem := setDefs[0].Element
		if ret := setElem.Extra.Mores[java.MethodReturnType]; ret != "void" {
			t.Errorf("setId expected return void, got %v", ret)
		}
	})

	// 断言 8 & 9: 内部类 EntityMeta 及字段 tableName
	t.Run("Verify Nested Class EntityMeta", func(t *testing.T) {
		classQN := "com.example.base.AbstractBaseEntity.EntityMeta"
		classDefs := test.FindDefinitionsByQN(fCtx, classQN)
		if len(classDefs) == 0 {
			t.Fatalf("Nested class EntityMeta not found")
		}

		classElem := classDefs[0].Element
		if !contains(classElem.Extra.Modifiers, "static") {
			t.Error("Should be static")
		}

		// 验证内部字段 tableName 的递归 QN
		fieldQN := "com.example.base.AbstractBaseEntity.EntityMeta.tableName"
		fieldDefs := test.FindDefinitionsByQN(fCtx, fieldQN)
		if len(fieldDefs) == 0 {
			t.Fatalf("Field tableName not found in nested class")
		}

		fieldElem := fieldDefs[0].Element
		if tpe := fieldElem.Extra.Mores[java.FieldRawType]; tpe != "String" {
			t.Errorf("tableName expected String, got %v", tpe)
		}
	})
}

func TestJavaCollector_BaseClassHierarchy(t *testing.T) {
	// 1. 获取测试文件路径
	filePath := test.GetTestFilePath(filepath.Join("collector", "base", "BaseClass.java"))

	// 2. 解析与收集
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	// 断言 1 & 2: 验证 BaseClass (Abstract, Annotations, Interfaces)
	t.Run("Verify BaseClass Metadata", func(t *testing.T) {
		qn := "com.example.base.BaseClass"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("BaseClass not found")
		}
		elem := defs[0].Element

		// 断言 1: 注解验证
		expectedAnnos := []string{"@Deprecated", "@SuppressWarnings(\"unused\")"}
		for _, anno := range expectedAnnos {
			if !contains(elem.Extra.Annotations, anno) {
				t.Errorf("BaseClass missing annotation: %s", anno)
			}
		}

		// 断言 2: Abstract 属性与接口
		if isAbs, ok := elem.Extra.Mores[java.ClassIsAbstract].(bool); !ok || !isAbs {
			t.Error("Expected java.class.is_abstract to be true")
		}

		interfaces, ok := elem.Extra.Mores[java.ClassImplementedInterfaces].([]string)
		if !ok || !contains(interfaces, "Serializable") {
			t.Errorf("Expected Serializable interface, got %v", elem.Extra.Mores[java.ClassImplementedInterfaces])
		}
	})

	// 断言 3 & 4: 验证 FinalClass (Final, SuperClass, Multiple Interfaces, Location)
	t.Run("Verify FinalClass Metadata", func(t *testing.T) {
		qn := "com.example.base.FinalClass"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("FinalClass not found")
		}
		elem := defs[0].Element

		// 断言 4: Kind 验证
		if elem.Kind != model.Class {
			t.Errorf("Expected Kind CLASS, got %s", elem.Kind)
		}

		// 断言 4: 位置信息验证 (FinalClass 在 BaseClass 之后，大致在第 11 行左右)
		if elem.Location.StartLine < 5 {
			t.Errorf("FinalClass StartLine seems incorrect: %d", elem.Location.StartLine)
		}

		// 断言 3: Final 属性
		if isFinal, ok := elem.Extra.Mores[java.ClassIsFinal].(bool); !ok || !isFinal {
			t.Error("Expected java.class.is_final to be true")
		}

		// 断言 3: 父类验证
		super, _ := elem.Extra.Mores[java.ClassSuperClass].(string)
		if !strings.Contains(super, "BaseClass") {
			t.Errorf("Expected super class BaseClass, got %q", super)
		}

		// 断言 3: 多接口验证
		interfaces, _ := elem.Extra.Mores[java.ClassImplementedInterfaces].([]string)
		if len(interfaces) < 2 || !contains(interfaces, "Cloneable") || !contains(interfaces, "Runnable") {
			t.Errorf("Expected multiple interfaces (Cloneable, Runnable), got %v", interfaces)
		}
	})

	// 断言 5: 验证 FinalClass.run() 函数的注解
	t.Run("Verify FinalClass.run() Annotations", func(t *testing.T) {
		qn := "com.example.base.FinalClass.run()"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method run() not found")
		}
		elem := defs[0].Element

		if !contains(elem.Extra.Annotations, "@Override") {
			t.Error("Method run() missing @Override annotation")
		}
	})
}

func TestJavaCollector_CallbackManager(t *testing.T) {
	// 1. 获取测试文件路径
	filePath := test.GetTestFilePath(filepath.Join("collector", "base", "CallbackManager.java"))

	// 2. 解析源码与运行 Collector
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	// 验证 1: 验证方法内部定义的局部类 LocalValidator
	t.Run("Verify Local Class", func(t *testing.T) {
		// 根据你的 Collector 实现，局部类应该在方法 QN 下
		qn := "com.example.base.CallbackManager.register().LocalValidator"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Local class LocalValidator not found at %s", qn)
		}

		elem := defs[0].Element
		if elem.Kind != model.Class {
			t.Errorf("Expected Kind CLASS, got %s", elem.Kind)
		}

		// 验证局部类内部的方法
		methodQN := qn + ".isValid()"
		methodDefs := test.FindDefinitionsByQN(fCtx, methodQN)
		if len(methodDefs) == 0 {
			t.Errorf("Method isValid() not found in local class")
		}
		if methodDefs[0].Element.Extra.Mores[java.MethodParameters] != nil {
			t.Errorf("Method isValid() found params")
		}
	})

	// 验证 2: 验证变量 r
	t.Run("Verify Variable r", func(t *testing.T) {
		qn := "com.example.base.CallbackManager.register().r"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Variable r not found at %s", qn)
		}

		elem := defs[0].Element
		if tpe := elem.Extra.Mores[java.VariableRawType]; tpe != "Runnable" {
			t.Errorf("Expected type Runnable, got %v", tpe)
		}
	})

	// 验证 3: 验证匿名内部类及其方法 run()
	t.Run("Verify Anonymous Inner Class and Run Method", func(t *testing.T) {
		// 修正路径：anonymousClass$1 现在应该正确嵌套了 run()
		anonQN := "com.example.base.CallbackManager.register().anonymousClass$1"
		runQN := anonQN + ".run()"

		runDefs := test.FindDefinitionsByQN(fCtx, runQN)
		if len(runDefs) == 0 {
			t.Fatalf("Method run() not found at expected QN: %s", runQN)
		}

		elem := runDefs[0].Element
		if !contains(elem.Extra.Annotations, "@Override") {
			t.Error("Method run() missing @Override")
		}
	})
}

func TestJavaCollector_ConfigService(t *testing.T) {
	// 1. 获取测试文件路径
	filePath := test.GetTestFilePath(filepath.Join("collector", "base", "ConfigService.java"))

	// 2. 解析源码与运行 Collector
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, false)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	// 验证 1: 变长参数 (Object...) 与 数组参数 (String[])
	t.Run("Verify Variadic and Array Parameters", func(t *testing.T) {
		// 注意：QN 内部的参数类型应反映原始定义
		qn := "com.example.base.ConfigService.updateConfigs(String[],Object...)"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method updateConfigs not found with expected signature QN: %s", qn)
		}

		elem := defs[0].Element
		params, ok := elem.Extra.Mores[java.MethodParameters].([]string)
		if !ok || len(params) != 2 {
			t.Fatalf("Expected 2 parameters, got %v", params)
		}

		// 验证数组参数
		if !strings.Contains(params[0], "String[]") {
			t.Errorf("Expected first param to be String[], got %s", params[0])
		}

		// 验证变长参数
		if !strings.Contains(params[1], "Object...") {
			t.Errorf("Expected second param to be Object..., got %s", params[1])
		}
	})

	// 验证 2: 复杂多属性注解
	t.Run("Verify Complex Annotations", func(t *testing.T) {
		qn := "com.example.base.ConfigService.legacyMethod()"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method legacyMethod not found")
		}

		elem := defs[0].Element
		annos := elem.Extra.Annotations

		// 验证 @SuppressWarnings 的数组格式
		foundSuppressed := false
		for _, a := range annos {
			if strings.Contains(a, "@SuppressWarnings") && strings.Contains(a, "\"unchecked\"") && strings.Contains(a, "\"rawtypes\"") {
				foundSuppressed = true
				break
			}
		}
		if !foundSuppressed {
			t.Errorf("Could not find complete @SuppressWarnings annotation, got: %v", annos)
		}

		// 验证 @Deprecated 的多属性 (since, forRemoval)
		foundDeprecated := false
		for _, a := range annos {
			if strings.Contains(a, "@Deprecated") && strings.Contains(a, "since = \"1.2\"") && strings.Contains(a, "forRemoval = true") {
				foundDeprecated = true
				break
			}
		}
		if !foundDeprecated {
			t.Errorf("Could not find detailed @Deprecated annotation, got: %v", annos)
		}
	})

	t.Run("Verify Specific Parameters", func(t *testing.T) {
		// 验证 keys
		keysQN := "com.example.base.ConfigService.updateConfigs(String[],Object...).keys"
		if len(test.FindDefinitionsByQN(fCtx, keysQN)) == 0 {
			t.Errorf("Variable 'keys' not found")
		}

		// 验证 values
		valuesQN := "com.example.base.ConfigService.updateConfigs(String[],Object...).values"
		vDefs := test.FindDefinitionsByQN(fCtx, valuesQN)
		if len(vDefs) == 0 {
			t.Fatalf("Variable 'values' not found")
		}

		vElem := vDefs[0].Element
		if tpe := vElem.Extra.Mores[java.VariableRawType]; tpe != "Object..." {
			t.Errorf("Expected type Object..., got %v", tpe)
		}
	})
}

func TestJavaCollector_DataProcessor(t *testing.T) {
	// 1. 获取测试文件路径
	filePath := test.GetTestFilePath(filepath.Join("collector", "base", "DataProcessor.java"))

	// 2. 解析源码与运行 Collector
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, false)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	// 验证 1: 接口定义、多继承与泛型
	t.Run("Verify Interface Heritage and Generics", func(t *testing.T) {
		qn := "com.example.base.DataProcessor"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Interface DataProcessor not found")
		}

		elem := defs[0].Element
		// 验证接口继承
		ifaces, _ := elem.Extra.Mores[java.InterfaceImplementedInterfaces].([]string)
		expectedIfaces := []string{"Runnable", "AutoCloseable"}
		for _, expected := range expectedIfaces {
			if !contains(ifaces, expected) {
				t.Errorf("Expected interface %s not found in %v", expected, ifaces)
			}
		}

		// 验证签名中的泛型参数 (T extends AbstractBaseEntity<?>)
		if !strings.Contains(elem.Signature, "<T extends AbstractBaseEntity<?>>") {
			t.Errorf("Signature missing generics: %s", elem.Signature)
		}
	})

	// 验证 2: 方法的 Throws 异常
	t.Run("Verify Method Throws", func(t *testing.T) {
		// 注意：泛型 T 在 QN 中按原样提取
		qn := "com.example.base.DataProcessor.processAll(String)"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method processAll not found")
		}

		elem := defs[0].Element
		throws, _ := elem.Extra.Mores[java.MethodThrowsTypes].([]string)

		expectedThrows := []string{"RuntimeException", "Exception"}
		if len(throws) != 2 {
			t.Fatalf("Expected 2 throws types, got %v", throws)
		}
		for i, e := range expectedThrows {
			if throws[i] != e {
				t.Errorf("Expected throw %s, got %s", e, throws[i])
			}
		}
	})

	// 验证 3: Java 8 Default 方法修饰符
	t.Run("Verify Default Method", func(t *testing.T) {
		qn := "com.example.base.DataProcessor.stop()"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method stop not found")
		}

		elem := defs[0].Element
		// 验证是否包含 default 关键字
		if !contains(elem.Extra.Modifiers, "default") {
			t.Errorf("Method stop should have 'default' modifier, got %v", elem.Extra.Modifiers)
		}

		// 验证 Signature 是否正确包含 default
		if !strings.HasPrefix(elem.Signature, "default void stop()") {
			t.Errorf("Signature prefix incorrect: %s", elem.Signature)
		}
	})
}

func TestJavaCollector_User(t *testing.T) {
	// 1. 初始化解析环境
	filePath := test.GetTestFilePath(filepath.Join("collector", "base", "User.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证静态导入 (Static Imports)
	t.Run("Verify Static Imports", func(t *testing.T) {
		// 静态导入 DAYS
		if imp, ok := fCtx.Imports["DAYS"]; !ok {
			t.Error("Static import 'DAYS' not found in FileContext")
		} else {
			entry := imp[0]
			if entry.Kind != model.Constant {
				t.Errorf("Expected DAYS to be Kind Constant, got %s", entry.Kind)
			}
			if entry.RawImportPath != "java.util.concurrent.TimeUnit.DAYS" {
				t.Errorf("Incorrect path for DAYS: %s", entry.RawImportPath)
			}
		}
	})

	// 2. 验证静态常量 (Static Final Field)
	t.Run("Verify Constant Field DEFAULT_ID", func(t *testing.T) {
		qn := "com.example.base.User.DEFAULT_ID"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Field DEFAULT_ID not found")
		}
		elem := defs[0].Element

		// 验证元数据中的常量标记
		if isConst, _ := elem.Extra.Mores[java.FieldIsConstant].(bool); !isConst {
			t.Error("DEFAULT_ID should be identified as a Constant (static + final)")
		}
		if fType := elem.Extra.Mores[java.FieldRawType].(string); fType != "String" {
			t.Errorf("Expected field type String, got %s", fType)
		}
	})

	// 3. 验证静态内部类 (Nested Class)
	t.Run("Verify Inner Class AddonInfo", func(t *testing.T) {
		qn := "com.example.base.User.AddonInfo"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Inner class AddonInfo not found")
		}
		elem := defs[0].Element
		if elem.Kind != model.Class {
			t.Errorf("Expected Kind Class, got %s", elem.Kind)
		}

		// 验证内部类字段的递归 QN
		fieldQN := "com.example.base.User.AddonInfo.otherName"
		if len(test.FindDefinitionsByQN(fCtx, fieldQN)) == 0 {
			t.Errorf("Field otherName in inner class not found with QN: %s", fieldQN)
		}
	})

	// 4. 验证 if 块产生的作用域 (ScopeBlock)
	t.Run("Verify If Blocks in chooseUnit", func(t *testing.T) {
		// chooseUnit 方法内部有多个 if 块。
		// 根据你的 applyUniqueQN 逻辑，它们应该被命名为 block$1, block$2 等
		methodQN := "com.example.base.User.AddonInfo.chooseUnit(long)"

		// 验证 block$1 (第一个 if 分支的内容)
		block1QN := methodQN + ".block$1"
		defs := test.FindDefinitionsByQN(fCtx, block1QN)
		if len(defs) == 0 {
			t.Fatalf("First if-block (block$1) not found in chooseUnit")
		}

		elem := defs[0].Element
		if elem.Kind != model.ScopeBlock {
			t.Errorf("Expected ScopeBlock, got %s", elem.Kind)
		}

		// 验证 block$2 (第二个 if 分支)
		block2QN := methodQN + ".block$2"
		if len(test.FindDefinitionsByQN(fCtx, block2QN)) == 0 {
			t.Error("Second if-block (block$2) not found")
		}
	})
}

func TestJavaCollector_UserServiceImpl(t *testing.T) {
	// 1. 初始化解析环境
	filePath := test.GetTestFilePath(filepath.Join("collector", "base", "UserServiceImpl.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	// 2. 执行 Collector
	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	test.PrintCodeElements(fCtx)

	// --- 断言开始 ---

	// 1. 验证类声明：包含注解、复杂的泛型继承与实现
	t.Run("Verify UserServiceImpl Class Definition", func(t *testing.T) {
		qn := "com.example.base.UserServiceImpl"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Class UserServiceImpl not found")
		}
		elem := defs[0].Element

		// 验证注解
		if !contains(elem.Extra.Annotations, "@Loggable") {
			t.Errorf("Expected annotation @Loggable, got %v", elem.Extra.Annotations)
		}

		// 验证泛型父类
		if super := elem.Extra.Mores[java.ClassSuperClass].(string); super != "AbstractBaseEntity<String>" {
			t.Errorf("Expected superclass AbstractBaseEntity<String>, got %s", super)
		}

		// 验证泛型接口实现 (DataProcessor<AbstractBaseEntity<String>>)
		ifaces, ok := elem.Extra.Mores[java.ClassImplementedInterfaces].([]string)
		if !ok || !contains(ifaces, "DataProcessor<AbstractBaseEntity<String>>") {
			t.Errorf("Expected interface DataProcessor<AbstractBaseEntity<String>> in %v", ifaces)
		}

		// 验证完整 Signature
		expectedSig := "public class UserServiceImpl extends AbstractBaseEntity<String> implements DataProcessor<AbstractBaseEntity<String>>"
		if elem.Signature != expectedSig {
			t.Errorf("Signature mismatch.\nGot: %s\nExp: %s", elem.Signature, expectedSig)
		}
	})

	// 2. 验证泛型方法：包含 Override 注解、泛型返回值、Throws 异常
	t.Run("Verify Method processAll", func(t *testing.T) {
		qn := "com.example.base.UserServiceImpl.processAll(String)"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method processAll(String) not found")
		}
		elem := defs[0].Element

		// 验证泛型返回值
		if ret := elem.Extra.Mores[java.MethodReturnType].(string); ret != "List<AbstractBaseEntity<String>>" {
			t.Errorf("Expected return type List<AbstractBaseEntity<String>>, got %s", ret)
		}

		// 验证 Throws 声明
		throws, ok := elem.Extra.Mores[java.MethodThrowsTypes].([]string)
		if !ok || !contains(throws, "RuntimeException") {
			t.Errorf("Expected throws RuntimeException, got %v", throws)
		}

		// 验证方法 Signature (应包含 public 和 throws)
		if !strings.Contains(elem.Signature, "public List<AbstractBaseEntity<String>> processAll(String batchId)") {
			t.Errorf("Signature should contain access modifier and generic return type, got: %s", elem.Signature)
		}
		if !strings.Contains(elem.Signature, "throws RuntimeException") {
			t.Errorf("Signature should contain throws clause, got: %s", elem.Signature)
		}
	})

	// 3. 验证方法体内的局部变量 (Local Variables)
	t.Run("Verify Local Variables in processAll", func(t *testing.T) {
		methodQN := "com.example.base.UserServiceImpl.processAll(String)"

		// 验证 results 变量
		resultsQN := methodQN + ".results"
		rDefs := test.FindDefinitionsByQN(fCtx, resultsQN)
		if len(rDefs) == 0 {
			t.Errorf("Local variable 'results' not found with QN: %s", resultsQN)
		} else {
			vType := rDefs[0].Element.Extra.Mores[java.VariableRawType].(string)
			if vType != "List<AbstractBaseEntity<String>>" {
				t.Errorf("Incorrect type for results: %s", vType)
			}
		}

		// 验证 converted 变量 (Cast 表达式后的变量)
		convertedQN := methodQN + ".converted"
		cDefs := test.FindDefinitionsByQN(fCtx, convertedQN)
		if len(cDefs) == 0 {
			t.Errorf("Local variable 'converted' not found")
		} else {
			vType := cDefs[0].Element.Extra.Mores[java.VariableRawType].(string)
			if vType != "String" {
				t.Errorf("Expected type String for 'converted', got %s", vType)
			}
		}
	})

	// 4. 验证构造函数及其内的 Field Access (隐式验证 QN 深度)
	t.Run("Verify Constructor and Implicit Logic", func(t *testing.T) {
		// 构造函数 QN 通常以类名命名
		ctorQN := "com.example.base.UserServiceImpl.UserServiceImpl()"
		defs := test.FindDefinitionsByQN(fCtx, ctorQN)
		if len(defs) == 0 {
			t.Fatalf("Constructor UserServiceImpl() not found")
		}

		elem := defs[0].Element
		if !elem.Extra.Mores[java.MethodIsConstructor].(bool) {
			t.Error("Should be identified as a constructor")
		}
	})
}

// 判断 slice 是否包含 string
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
