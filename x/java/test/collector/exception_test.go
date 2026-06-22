package collector

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

func TestJavaCollector_NotificationException(t *testing.T) {
	// 1. 初始化解析环境
	filePath := test.GetTestFilePath(filepath.Join("collector", "exception", "NotificationException.java"))
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

	// 1. 验证类的继承关系 (EXTEND)
	t.Run("Verify Exception Inheritance", func(t *testing.T) {
		qn := "com.example.base.exception.NotificationException"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Class NotificationException not found")
		}
		elem := defs[0].Element

		// 验证 SuperClass 字段 (对应 java.class.superclass)
		super, ok := elem.Extra.Mores[constants.ClassSuperClass].(string)
		if !ok || super != "Exception" {
			t.Errorf("Expected superclass 'Exception', got '%v'", super)
		}
	})

	// 2. 验证序列化常量 (Field)
	t.Run("Verify serialVersionUID Field", func(t *testing.T) {
		qn := "com.example.base.exception.NotificationException.serialVersionUID"
		defs := test.FindDefinitionsByQN(fCtx, qn)
		if len(defs) == 0 {
			t.Fatalf("Field serialVersionUID not found")
		}
		elem := defs[0].Element

		// 验证常量属性 (static + final)
		isConstant := elem.Extra.Mores[constants.FieldIsConstant].(bool)
		if !isConstant {
			t.Error("serialVersionUID should be identified as a constant")
		}

		fieldType := elem.Extra.Mores[constants.FieldRawType].(string)
		if fieldType != "long" {
			t.Errorf("Expected type long, got %s", fieldType)
		}
	})

	// 3. 验证多个构造函数 (Constructor Overloading)
	t.Run("Verify Overloaded Constructors", func(t *testing.T) {
		// 构造函数 A: (String, Throwable)
		qnA := "com.example.base.exception.NotificationException.NotificationException(String,Throwable)"
		defsA := test.FindDefinitionsByQN(fCtx, qnA)
		if len(defsA) == 0 {
			t.Fatalf("Constructor (String, Throwable) not found")
		}
		if !defsA[0].Element.Extra.Mores[constants.MethodIsConstructor].(bool) {
			t.Error("Should be marked as constructor")
		}

		// 构造函数 B: (ErrorCode)
		qnB := "com.example.base.exception.NotificationException.NotificationException(ErrorCode)"
		defsB := test.FindDefinitionsByQN(fCtx, qnB)
		if len(defsB) == 0 {
			t.Fatalf("Constructor (ErrorCode) not found")
		}

		// 验证参数元数据
		params, _ := defsB[0].Element.Extra.Mores[constants.MethodParameters].([]string)
		if len(params) != 1 || !strings.Contains(params[0], "ErrorCode code") {
			t.Errorf("Incorrect parameters metadata: %v", params)
		}
	})
}
