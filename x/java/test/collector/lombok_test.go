package collector

import (
	"path/filepath"
	"testing"

	"github.com/CodMac/arch-lens-dep-analyer/model"
	"github.com/CodMac/arch-lens-dep-analyer/x/java"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/constants"
	"github.com/CodMac/arch-lens-dep-analyer/x/java/test"
)

// TestJavaCollector_LombokBuilderComprehensive 测试 @Builder 注解综合场景
func TestJavaCollector_LombokBuilderComprehensive(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "lombok", "BuilderComprehensive.java"))
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

	t.Run("Verify_Builder_Class", func(t *testing.T) {
		builderQN := "com.example.lombok.BuilderComprehensive.Builder"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == builderQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("Builder class should be marked as IsFormSugar")
				}
				if entry.Element.Kind != model.Class {
					t.Errorf("Builder should be a CLASS, got %v", entry.Element.Kind)
				}
				break
			}
		}
		if !found {
			t.Errorf("Builder class NOT found. QN: %s", builderQN)
		}
	})

	t.Run("Verify_Builder_Static_Method", func(t *testing.T) {
		builderMethodQN := "com.example.lombok.BuilderComprehensive.builder()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == builderMethodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("builder() method should be marked as IsFormSugar")
				}
				if entry.Element.Name != "builder" {
					t.Errorf("Method should be named 'builder', got '%s'", entry.Element.Name)
				}
				break
			}
		}
		if !found {
			t.Errorf("builder() method NOT found. QN: %s", builderMethodQN)
		}
	})

	t.Run("Verify_Builder_Setter_Methods", func(t *testing.T) {
		expectedSetters := []struct {
			name  string
			param string
		}{
			{"host", "String"},
			{"port", "int"},
			{"secure", "boolean"},
			{"username", "String"},
			{"password", "String"},
		}

		for _, expected := range expectedSetters {
			setterQN := "com.example.lombok.BuilderComprehensive.Builder." + expected.name + "(" + expected.param + ")"
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.QualifiedName == setterQN {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Builder setter %s should be marked as IsFormSugar", expected.name)
					}
					break
				}
			}
			if !found {
				t.Errorf("Builder setter %s NOT found", expected.name)
			}
		}
	})

	t.Run("Verify_Build_Method", func(t *testing.T) {
		buildMethodQN := "com.example.lombok.BuilderComprehensive.Builder.build()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == buildMethodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("build() method should be marked as IsFormSugar")
				}
				if entry.Element.Name != "build" {
					t.Errorf("Method should be named 'build', got '%s'", entry.Element.Name)
				}
				break
			}
		}
		if !found {
			t.Errorf("build() method NOT found. QN: %s", buildMethodQN)
		}
	})

	t.Run("Verify_NoArgsConstructor", func(t *testing.T) {
		constructorQN := "com.example.lombok.BuilderComprehensive.BuilderComprehensive()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == constructorQN {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("No args constructor NOT found. QN: %s", constructorQN)
		}
	})

	t.Run("Verify_AllArgsConstructor", func(t *testing.T) {
		constructorQN := "com.example.lombok.BuilderComprehensive.BuilderComprehensive(String,int,boolean,String,String)"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == constructorQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("All args constructor should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("All args constructor NOT found. QN: %s", constructorQN)
		}
	})

	t.Run("Verify_Nested_Config_Builder_Class", func(t *testing.T) {
		builderQN := "com.example.lombok.BuilderComprehensive.Config.Builder"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == builderQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("Nested Builder class should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("Nested Builder class NOT found. QN: %s", builderQN)
		}
	})

	t.Run("Verify_Nested_Config_Builder_Static_Method", func(t *testing.T) {
		builderMethodQN := "com.example.lombok.BuilderComprehensive.Config.builder()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == builderMethodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("Nested builder() method should be marked as IsFormSugar")
				}
				if entry.Element.Name != "builder" {
					t.Errorf("Method should be named 'builder', got '%s'", entry.Element.Name)
				}
				break
			}
		}
		if !found {
			t.Errorf("Nested builder() method NOT found. QN: %s", builderMethodQN)
		}
	})

	t.Run("Verify_Nested_Config_Builder_Setter_Methods", func(t *testing.T) {
		expectedSetters := []struct {
			name  string
			param string
		}{
			{"database", "String"},
			{"maxConnections", "int"},
			{"timeout", "long"},
		}

		for _, expected := range expectedSetters {
			setterQN := "com.example.lombok.BuilderComprehensive.Config.Builder." + expected.name + "(" + expected.param + ")"
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.QualifiedName == setterQN {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Nested Builder setter %s should be marked as IsFormSugar", expected.name)
					}
					break
				}
			}
			if !found {
				t.Errorf("Nested Builder setter %s NOT found", expected.name)
			}
		}
	})

	t.Run("Verify_Nested_Config_Build_Method", func(t *testing.T) {
		buildMethodQN := "com.example.lombok.BuilderComprehensive.Config.Builder.build()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == buildMethodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("Nested build() method should be marked as IsFormSugar")
				}
				if entry.Element.Name != "build" {
					t.Errorf("Method should be named 'build', got '%s'", entry.Element.Name)
				}
				break
			}
		}
		if !found {
			t.Errorf("Nested build() method NOT found. QN: %s", buildMethodQN)
		}
	})
}

// TestJavaCollector_LombokConstructorComprehensive 测试构造器相关注解综合场景
func TestJavaCollector_LombokConstructorComprehensive(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "lombok", "ConstructorComprehensive.java"))
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

	t.Run("Verify_NoArgsConstructor", func(t *testing.T) {
		constructorQN := "com.example.lombok.ConstructorComprehensive.ConstructorComprehensive()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == constructorQN {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("No args constructor NOT found. QN: %s", constructorQN)
		}
	})

	t.Run("Verify_AllArgsConstructor", func(t *testing.T) {
		constructorQN := "com.example.lombok.ConstructorComprehensive.ConstructorComprehensive(String,int)"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == constructorQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("All args constructor should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("All args constructor NOT found. QN: %s", constructorQN)
		}
	})

	t.Run("Verify_RequiredArgsConstructor", func(t *testing.T) {
		constructorQN := "com.example.lombok.ConstructorComprehensive.RequiredArgsConstructor.RequiredArgsConstructor(String)"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == constructorQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("Required args constructor should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("Required args constructor NOT found. QN: %s", constructorQN)
		}
	})

	t.Run("Verify_InnerClass_Constructor", func(t *testing.T) {
		constructorQN := "com.example.lombok.ConstructorComprehensive.RequiredArgsConstructor.InnerClass.InnerClass(int)"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == constructorQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("Inner class constructor should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("Inner class constructor NOT found. QN: %s", constructorQN)
		}
	})
}

// TestJavaCollector_LombokDataComprehensive 测试 @Data 注解综合场景
func TestJavaCollector_LombokDataComprehensive(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "lombok", "DataComprehensive.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	t.Run("Verify_Getters", func(t *testing.T) {
		expectedGetters := []string{"getName", "getAge", "isActive", "getEmail"}
		for _, expectedName := range expectedGetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.DataComprehensive" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Getter %s should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Getter %s NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_Setters", func(t *testing.T) {
		expectedSetters := []string{"setName", "setAge", "setActive", "setEmail"}
		for _, expectedName := range expectedSetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.DataComprehensive" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Setter %s should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Setter %s NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_Default_Constructor", func(t *testing.T) {
		constructorQN := "com.example.lombok.DataComprehensive.DataComprehensive()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == constructorQN {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Default constructor NOT found. QN: %s", constructorQN)
		}
	})

	t.Run("Verify_Equals_Method", func(t *testing.T) {
		methodQN := "com.example.lombok.DataComprehensive.equals(Object)"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == methodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("equals() method should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("equals() method NOT found. QN: %s", methodQN)
		}
	})

	t.Run("Verify_HashCode_Method", func(t *testing.T) {
		methodQN := "com.example.lombok.DataComprehensive.hashCode()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == methodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("hashCode() method should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("hashCode() method NOT found. QN: %s", methodQN)
		}
	})

	t.Run("Verify_ToString_Method", func(t *testing.T) {
		methodQN := "com.example.lombok.DataComprehensive.toString()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == methodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("toString() method should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("toString() method NOT found. QN: %s", methodQN)
		}
	})
}

// TestJavaCollector_LombokEdgeCases 测试边界情况
func TestJavaCollector_LombokEdgeCases(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "lombok", "EdgeCases.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	t.Run("Verify_Primitive-Type_Getters", func(t *testing.T) {
		expectedGetters := []string{
			"getPrimitiveInt", "getPrimitiveLong", "isPrimitiveBoolean", "getPrimitiveDouble",
		}

		for _, expectedName := range expectedGetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.EdgeCases" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Primitive getter %s should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Primitive getter %s NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_Wrapper_Type_Getters", func(t *testing.T) {
		expectedGetters := []string{
			"getWrapperInt", "getWrapperLong", "isWrapperBoolean", "getWrapperDouble",
		}

		for _, expectedName := range expectedGetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.EdgeCases" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Wrapper getter %s should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Wrapper getter %s NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_String_Getters", func(t *testing.T) {
		expectedGetters := []string{"getStringField"}

		for _, expectedName := range expectedGetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.EdgeCases" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("String getter %s should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("String getter %s NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_Array_Getters", func(t *testing.T) {
		expectedGetters := []string{"getIntArray", "getStringArray"}

		for _, expectedName := range expectedGetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.EdgeCases" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Array getter %s should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Array getter %s NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_Transient_Field_Getters", func(t *testing.T) {
		transientGetters := []string{"getTransientField"}

		for _, expectedName := range transientGetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.EdgeCases" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Transient field getter %s should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Transient field getter %s NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_No_Static_Field_Getters", func(t *testing.T) {
		// 静态字段不应该生成Lombok的getter/setter
		staticGetter := "getStaticField"
		for _, entry := range fCtx.Definitions {
			if entry.Element.Kind == model.Method &&
				entry.Element.Name == staticGetter &&
				entry.ParentQN == "com.example.lombok.EdgeCases" &&
				entry.Element.IsFormSugar {
				t.Errorf("Static field should NOT have Lombok-generated getter")
			}
		}
	})

	t.Run("Verify_Primitive_Type_Setters", func(t *testing.T) {
		expectedSetters := []string{
			"setPrimitiveInt", "setPrimitiveLong", "setPrimitiveBoolean", "setPrimitiveDouble",
		}

		for _, expectedName := range expectedSetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.EdgeCases" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Primitive setter %s should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Primitive setter %s NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_Wrapper_Type_Setters", func(t *testing.T) {
		expectedSetters := []string{
			"setWrapperInt", "setWrapperLong", "setWrapperBoolean", "setWrapperDouble",
		}

		for _, expectedName := range expectedSetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.EdgeCases" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Wrapper setter %s should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Wrapper setter %s NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_Default_Constructor", func(t *testing.T) {
		constructorQN := "com.example.lombok.EdgeCases.EdgeCases()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == constructorQN {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Default constructor NOT found. QN: %s", constructorQN)
		}
	})

	t.Run("Verify_ToString_Method", func(t *testing.T) {
		methodQN := "com.example.lombok.EdgeCases.toString()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == methodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("toString() method should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("toString() method NOT found. QN: %s", methodQN)
		}
	})
}

// TestJavaCollector_LombokEqualsAndHashCode 测试 @EqualsAndHashCode 注解
func TestJavaCollector_LombokEqualsAndHashCode(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "lombok", "EqualsAndHashCodeExample.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	t.Run("Verify_Equals_Method", func(t *testing.T) {
		methodQN := "com.example.lombok.EqualsAndHashCodeExample.equals(Object)"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == methodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("equals() method should be marked as IsFormSugar")
				}
				if entry.Element.Name != "equals" {
					t.Errorf("Method name should be 'equals', got '%s'", entry.Element.Name)
				}
				break
			}
		}
		if !found {
			t.Errorf("equals() method NOT found. QN: %s", methodQN)
		}
	})

	t.Run("Verify_HashCode_Method", func(t *testing.T) {
		methodQN := "com.example.lombok.EqualsAndHashCodeExample.hashCode()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == methodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("hashCode() method should be marked as IsFormSugar")
				}
				if entry.Element.Name != "hashCode" {
					t.Errorf("Method name should be 'hashCode', got '%s'", entry.Element.Name)
				}
				break
			}
		}
		if !found {
			t.Errorf("hashCode() method NOT found. QN: %s", methodQN)
		}
	})

	t.Run("Verify_CanEqual_Method", func(t *testing.T) {
		methodQN := "com.example.lombok.EqualsAndHashCodeExample.canEqual(Object)"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == methodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("canEqual() method should be marked as IsFormSugar")
				}
				if entry.Element.Name != "canEqual" {
					t.Errorf("Method name should be 'canEqual', got '%s'", entry.Element.Name)
				}
				break
			}
		}
		if !found {
			t.Errorf("canEqual() method NOT found. QN: %s", methodQN)
		}
	})

	t.Run("Verify_Getters", func(t *testing.T) {
		expectedGetters := []string{"getId", "getName", "getVersion"}
		for _, expectedName := range expectedGetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.EqualsAndHashCodeExample" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Getter %s should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Getter %s NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_Setters", func(t *testing.T) {
		expectedSetters := []string{"setId", "setName", "setVersion"}
		for _, expectedName := range expectedSetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.EqualsAndHashCodeExample" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Setter %s should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Setter %s NOT found", expectedName)
			}
		}
	})
}

// TestJavaCollector_LombokGetterOnly 测试 @Getter 注解
func TestJavaCollector_LombokGetterOnly(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "lombok", "GetterOnly.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	t.Run("Verify_Getters_Only", func(t *testing.T) {
		expectedGetters := []string{
			"getName",  // String name字段的getter
			"getAge",   // int age字段的getter（final字段也可以有getter）
			"isActive", // boolean active字段的getter（使用 is 前缀）
		}

		for _, expectedName := range expectedGetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.GetterOnly" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Getter %s should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Getter %s NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_No_Setters", func(t *testing.T) {
		// 验证不应该有任何setter方法生成
		setterNames := []string{"setName", "setAge", "setActive"}
		for _, setterName := range setterNames {
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == setterName &&
					entry.ParentQN == "com.example.lombok.GetterOnly" &&
					entry.Element.IsFormSugar { // 只检查Lombok生成的setter
					t.Errorf("Lombok-generated setter %s should NOT exist for @Getter only class", setterName)
				}
			}
		}
	})

	t.Run("Verify_Final_Field_Getter", func(t *testing.T) {
		// 特别验证final字段也有getter
		finalFieldGetter := "getAge"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.Kind == model.Method &&
				entry.Element.Name == finalFieldGetter &&
				entry.ParentQN == "com.example.lombok.GetterOnly" {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("Final field getter %s should be marked as IsFormSugar", finalFieldGetter)
				}
				break
			}
		}
		if !found {
			t.Errorf("Final field getter %s NOT found", finalFieldGetter)
		}
	})

	t.Run("Verify_Boolean_Field_Getter_Prefix", func(t *testing.T) {
		// 验证boolean字段的getter使用 is 前缀
		booleanGetter := "isActive"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.Kind == model.Method &&
				entry.Element.Name == booleanGetter &&
				entry.ParentQN == "com.example.lombok.GetterOnly" {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("Boolean getter %s should be marked as IsFormSugar", booleanGetter)
				}

				// 特别检查应该是 isActive 而不是 getActive
				if entry.Element.Name != "isActive" {
					t.Errorf("Boolean field getter should use 'is' prefix, got '%s'", entry.Element.Name)
				}
				break
			}
		}
		if !found {
			t.Errorf("Boolean getter %s NOT found", booleanGetter)
		}
	})
}

// TestJavaCollector_LombokMultipleAnnotations 测试多注解组合场景
func TestJavaCollector_LombokMultipleAnnotations(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "lombok", "MultipleAnnotationsExample.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	t.Run("Verify_Data_Getters", func(t *testing.T) {
		expectedGetters := []string{"getId", "getName", "getValue"}
		for _, expectedName := range expectedGetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.MultipleAnnotationsExample" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Getter %s from @Data should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Getter %s from @Data NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_Data_Setters", func(t *testing.T) {
		expectedSetters := []string{"setId", "setName", "setValue"}
		for _, expectedName := range expectedSetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.MultipleAnnotationsExample" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Setter %s from @Data should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Setter %s from @Data NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_Data_ToString", func(t *testing.T) {
		methodQN := "com.example.lombok.MultipleAnnotationsExample.toString()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == methodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("toString() from @Data should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("toString() from @Data NOT found. QN: %s", methodQN)
		}
	})

	t.Run("Verify_Slf4j_Log_Field", func(t *testing.T) {
		// @Slf4j 应该生成一个名为 "log" 的日志字段
		logFieldQN := "com.example.lombok.MultipleAnnotationsExample.log"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == logFieldQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("'log' field from @Slf4j should be marked as IsFormSugar")
				}
				// 验证字段类型是否为 Logger
				if entry.Element.Extra != nil {
					fieldType, ok := entry.Element.Extra.Mores[constants.VariableRawType].(string)
					if ok && fieldType != "org.slf4j.Logger" {
						t.Errorf("'log' field should be of type 'org.slf4j.Logger', got '%s'", fieldType)
					}
				}
				break
			}
		}
		if !found {
			t.Errorf("'log' field from @Slf4j NOT found. QN: %s", logFieldQN)
		}
	})

	t.Run("Verify_Data_Default_Constructor", func(t *testing.T) {
		constructorQN := "com.example.lombok.MultipleAnnotationsExample.MultipleAnnotationsExample()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == constructorQN {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Default constructor from @Data NOT found. QN: %s", constructorQN)
		}
	})

	t.Run("Verify_Data_EqualsAndHashCode", func(t *testing.T) {
		equalsQN := "com.example.lombok.MultipleAnnotationsExample.equals(Object)"
		hashCodeQN := "com.example.lombok.MultipleAnnotationsExample.hashCode()"

		foundEquals := false
		foundHashCode := false

		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == equalsQN {
				foundEquals = true
				if !entry.Element.IsFormSugar {
					t.Errorf("equals() from @Data should be marked as IsFormSugar")
				}
			}
			if entry.Element.QualifiedName == hashCodeQN {
				foundHashCode = true
				if !entry.Element.IsFormSugar {
					t.Errorf("hashCode() from @Data should be marked as IsFormSugar")
				}
			}
		}

		if !foundEquals {
			t.Errorf("equals() from @Data NOT found. QN: %s", equalsQN)
		}
		if !foundHashCode {
			t.Errorf("hashCode() from @Data NOT found. QN: %s", hashCodeQN)
		}
	})
}

// TestJavaCollector_LombokSetterOnly 测试 @Setter 注解
func TestJavaCollector_LombokSetterOnly(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "lombok", "SetterOnly.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	t.Run("Verify_Setters_Only", func(t *testing.T) {
		expectedSetters := []string{
			"setName",   // String name字段的setter
			"setAge",    // int age字段的setter
			"setActive", // boolean active字段的setter
		}

		for _, expectedName := range expectedSetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.SetterOnly" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Setter %s should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Setter %s NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_No_Getters", func(t *testing.T) {
		// 验证不应该有任何getter方法生成（排除原生语法糖生成的）
		getterNames := []string{"getName", "getAge", "isActive"}
		for _, getterName := range getterNames {
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == getterName &&
					entry.ParentQN == "com.example.lombok.SetterOnly" &&
					entry.Element.IsFormSugar { // 只检查Lombok生成的getter
					t.Errorf("Lombok-generated getter %s should NOT exist for @Setter only class", getterName)
				}
			}
		}
	})
}

// TestJavaCollector_LombokSlf4j 测试 @Slf4j 注解
func TestJavaCollector_LombokSlf4j(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "lombok", "Slf4jExample.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	t.Run("Verify_Log_Field", func(t *testing.T) {
		logFieldQN := "com.example.lombok.Slf4jExample.log"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == logFieldQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("'log' field from @Slf4j should be marked as IsFormSugar")
				}
				// 验证字段名为 log
				if entry.Element.Name != "log" {
					t.Errorf("'log' field name should be 'log', got '%s'", entry.Element.Name)
				}
				break
			}
		}
		if !found {
			t.Errorf("'log' field from @Slf4j NOT found. QN: %s", logFieldQN)
		}
	})

	t.Run("Verify_Log_Field_Type", func(t *testing.T) {
		logFieldQN := "com.example.lombok.Slf4jExample.log"
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == logFieldQN {
				// 验证字段类型为 Logger
				if entry.Element.Extra != nil {
					fieldType, ok := entry.Element.Extra.Mores[constants.VariableRawType].(string)
					if !ok || fieldType == "" {
						t.Errorf("'log' field should have raw_type in Extra.Mores")
					}
					// 日志字段类型应该包含 "Logger"
					if fieldType != "org.slf4j.Logger" {
						t.Errorf("'log' field should be of type 'org.slf4j.Logger', got '%s'", fieldType)
					}
				}
				break
			}
		}
	})

	t.Run("Verify_Log_Field_Is_Field", func(t *testing.T) {
		logFieldQN := "com.example.lombok.Slf4jExample.log"
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == logFieldQN {
				// 验证生成的log是字段而不是方法
				if entry.Element.Kind != model.Field {
					t.Errorf("'log' from @Slf4j should be a FIELD, got %v", entry.Element.Kind)
				}
				break
			}
		}
	})
}

// TestJavaCollector_LombokToString 测试 @ToString 注解
func TestJavaCollector_LombokToString(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "lombok", "ToStringExample.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	t.Run("Verify_ToString_Method_Basic", func(t *testing.T) {
		methodQN := "com.example.lombok.ToStringExample.toString()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == methodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("toString() method from @ToString should be marked as IsFormSugar")
				}
				if entry.Element.Name != "toString" {
					t.Errorf("Method name should be 'toString', got '%s'", entry.Element.Name)
				}
				break
			}
		}
		if !found {
			t.Errorf("toString() method NOT found. QN: %s", methodQN)
		}
	})

	t.Run("Verify_ToString_Method_Detailed", func(t *testing.T) {
		methodQN := "com.example.lombok.ToStringDetailedExample.toString()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == methodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("toString() method from @ToString should be marked as IsFormSugar")
				}
				if entry.Element.Name != "toString" {
					t.Errorf("Method name should be 'toString', got '%s'", entry.Element.Name)
				}
				break
			}
		}
		if !found {
			t.Errorf("toString() method NOT found. QN: %s", methodQN)
		}
	})

	t.Run("Verify_No_Getters_Setters", func(t *testing.T) {
		// 验证 @ToString 只生成toString方法，不生成getter/setter
		// 对于 ToStringExample 类
		badMethods := []string{"getName", "setName", "getAge", "setAge", "getActive", "setActive"}
		for _, methodName := range badMethods {
			for _, entry := range fCtx.Definitions {
				if entry.Element.Name == methodName &&
					entry.ParentQN == "com.example.lombok.ToStringExample" &&
					entry.Element.IsFormSugar {
					t.Errorf("Lombok-generated %s should NOT exist for @ToString only class", methodName)
				}
			}
		}
	})
}

// TestJavaCollector_LombokValue 测试 @Value 注解
func TestJavaCollector_LombokValue(t *testing.T) {
	filePath := test.GetTestFilePath(filepath.Join("collector", "sugar", "lombok", "ValueExample.java"))
	rootNode, sourceBytes, err := test.GetJavaParser(t).ParseFile(filePath, false, true)
	if err != nil {
		t.Fatalf("Failed to parse file: %v", err)
	}

	collector := java.NewJavaCollector()
	fCtx, err := collector.CollectDefinitions(rootNode, filePath, sourceBytes)
	if err != nil {
		t.Fatalf("CollectDefinitions failed: %v", err)
	}

	t.Run("Verify_Getters", func(t *testing.T) {
		expectedGetters := []string{"getName", "getAge", "isActive"}
		for _, expectedName := range expectedGetters {
			found := false
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == expectedName &&
					entry.ParentQN == "com.example.lombok.ValueExample" {
					found = true
					if !entry.Element.IsFormSugar {
						t.Errorf("Getter %s from @Value should be marked as IsFormSugar", expectedName)
					}
					break
				}
			}
			if !found {
				t.Errorf("Getter %s from @Value NOT found", expectedName)
			}
		}
	})

	t.Run("Verify_No_Setters", func(t *testing.T) {
		// @Value 创建不可变对象，不应该有setter
		setterNames := []string{"setName", "setAge", "setActive"}
		for _, setterName := range setterNames {
			for _, entry := range fCtx.Definitions {
				if entry.Element.Kind == model.Method &&
					entry.Element.Name == setterName &&
					entry.ParentQN == "com.example.lombok.ValueExample" &&
					entry.Element.IsFormSugar {
					t.Errorf("Lombok-generated setter %s should NOT exist for @Value immutable class", setterName)
				}
			}
		}
	})

	t.Run("Verify_AllArgsConstructor", func(t *testing.T) {
		// @Value 生成全参构造器
		constructorQN := "com.example.lombok.ValueExample.ValueExample(String,int,boolean)"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == constructorQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("All args constructor from @Value should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("All args constructor from @Value NOT found. QN: %s", constructorQN)
		}
	})

	t.Run("Verify_Equals_Method", func(t *testing.T) {
		methodQN := "com.example.lombok.ValueExample.equals(Object)"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == methodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("equals() from @Value should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("equals() from @Value NOT found. QN: %s", methodQN)
		}
	})

	t.Run("Verify_HashCode_Method", func(t *testing.T) {
		methodQN := "com.example.lombok.ValueExample.hashCode()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == methodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("hashCode() from @Value should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("hashCode() from @Value NOT found. QN: %s", methodQN)
		}
	})

	t.Run("Verify_ToString_Method", func(t *testing.T) {
		methodQN := "com.example.lombok.ValueExample.toString()"
		found := false
		for _, entry := range fCtx.Definitions {
			if entry.Element.QualifiedName == methodQN {
				found = true
				if !entry.Element.IsFormSugar {
					t.Errorf("toString() from @Value should be marked as IsFormSugar")
				}
				break
			}
		}
		if !found {
			t.Errorf("toString() from @Value NOT found. QN: %s", methodQN)
		}
	})
}
