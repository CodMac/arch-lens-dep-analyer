package collector

import (
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestJavaCollector_EnumErrorCode(t *testing.T) {
	// 1. 初始化解析环境
	filePath := test.GetTestFilePath(filepath.Join("collector", "enum", "ErrorCode.java"))
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

	// 1. 验证枚举主体及其全限定名 (QN)
	t.Run("Verify Enum Entity", func(t *testing.T) {
		qn := "com.example.base.enum.ErrorCode"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Enum ErrorCode not found")
		}
		elem := defs[0].Element
		if elem.Kind != model.Enum {
			t.Errorf("Expected Kind ENUM, got %s", elem.Kind)
		}
	})

	// 2. 验证枚举常量及其参数 (使用 java.EnumArguments)
	t.Run("Verify Enum Constant Arguments", func(t *testing.T) {
		qn := "com.example.base.enum.ErrorCode.USER_NOT_FOUND"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Enum constant USER_NOT_FOUND not found")
		}

		elem := defs[0].Element
		// 验证枚举常量被识别为 Field (对应你的 identifyElement 逻辑)
		if elem.Kind != model.EnumConstant {
			t.Errorf("Expected Enum Constant to be Kind Field, got %s", elem.Kind)
		}

		// 核心验证：检查参数提取 (404, "User not found...")
		args, ok := elem.Extra.Mores[constants.EnumArguments].([]string)
		if !ok {
			t.Fatalf("Metadata key %s (EnumArguments) not found or wrong type", constants.EnumArguments)
		}

		if len(args) != 2 {
			t.Errorf("Expected 2 arguments, got %d", len(args))
		}
		if args[0] != "404" {
			t.Errorf("Expected first arg 404, got %s", args[0])
		}
	})

	// 3. 验证构造函数 (使用 java.MethodIsConstructor)
	t.Run("Verify Enum Constructor", func(t *testing.T) {
		// 构造函数 QN 包含参数类型
		qn := "com.example.base.enum.ErrorCode.ErrorCode(int,String)"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Enum constructor with QN %s not found", qn)
		}

		elem := defs[0].Element
		isCtor, ok := elem.Extra.Mores[constants.MethodIsConstructor].(bool)
		if !ok || !isCtor {
			t.Errorf("Expected %s to be true", constants.MethodIsConstructor)
		}
	})

	// 4. 验证成员方法及其返回值类型 (使用 java.MethodReturnType)
	t.Run("Verify Enum Member Methods", func(t *testing.T) {
		qn := "com.example.base.enum.ErrorCode.getMessage()"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Method getMessage() not found")
		}

		elem := defs[0].Element
		retType, ok := elem.Extra.Mores[constants.MethodReturnType].(string)
		if !ok || retType != "String" {
			t.Errorf("Expected return type String, got %v", retType)
		}
	})
}
