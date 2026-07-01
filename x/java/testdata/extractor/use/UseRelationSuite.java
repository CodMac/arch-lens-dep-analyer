package com.example.rel;

import java.util.List;

public class UseRelationSuite {

    public static final String CONSTANT = "FIXED";
    private int fieldVar = 10;

    public void testUseCases(int param) {
        // 1. 局部变量读取 (Local Variable Use)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).local
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.context_ast_kind": "binary_expression", "java.rel.raw_text": "local" }
        int local = 5;
        int result = local + 2;

        // 2. 成员变量读取 (Field Use - Explicit this)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.fieldVar
        // Mores: { "java.rel.use.receiver": "this", "java.rel.ast_kind": "identifier", "java.rel.context_ast_kind": "field_access", "java.rel.raw_text": "fieldVar" }
        int x = this.fieldVar;

        // 3. 隐式成员变量与参数读取
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).param
        // Target: com.example.rel.UseRelationSuite.CONSTANT
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.context_ast_kind": "binary_expression" }
        this.fieldVar = param + CONSTANT;

        // 4. 静态常量读取 (Static Constant Use)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.CONSTANT
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.context_ast_kind": "method_invocation", "java.rel.raw_text": "CONSTANT" }
        System.out.println(CONSTANT);

        // 5. 数组元素读取 (Array Access Element Use)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).args
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.context_ast_kind": "array_access", "java.rel.raw_text": "args" }
        String[] args = new String[]{"test"};
        String s = args[0];

        // 6. 作为方法调用实参读取 (Method Argument Use)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).local
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.context_ast_kind": "method_invocation", "java.rel.raw_text": "local" }
        genericMethod(local);

        // 7. 三元表达式中的读取 (Ternary Expression Operands)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).local
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.context_ast_kind": "ternary_expression", "java.rel.raw_text": "local" }
        int val = (local > 0) ? local : 0;

        // 8. 增强 for 循环中的集合读取
        List<String> list = List.of("A", "B");
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).list
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.context_ast_kind": "enhanced_for_statement", "java.rel.raw_text": "list" }
        for (String item : list) {
            // Source: com.example.rel.UseRelationSuite.testUseCases(int)
            // Target: com.example.rel.UseRelationSuite.testUseCases(int).item
            // Mores: { "java.rel.ast_kind": "identifier", "java.rel.context_ast_kind": "method_invocation", "java.rel.raw_text": "item" }
            System.out.println(item);
        }

        // 9. Lambda 捕获读取 (Variable Capture)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int).lambda$1
        // Target: com.example.rel.UseRelationSuite.fieldVar
        // Mores: { "java.rel.use.is_capture": true, "java.rel.ast_kind": "identifier", "java.rel.context_ast_kind": "method_invocation", "java.rel.raw_text": "fieldVar" }
        Runnable r = () -> {
            System.out.println(fieldVar);
        };

        // 10. 类型强制转换中的读取 (Cast Operand Use)
        // Source: com.example.rel.UseRelationSuite.testUseCases(int)
        // Target: com.example.rel.UseRelationSuite.testUseCases(int).obj
        // Mores: { "java.rel.ast_kind": "identifier", "java.rel.context_ast_kind": "cast_expression", "java.rel.raw_text": "obj" }
        Object obj = "string";
        String str = (String) obj;
    }

    public <T> void genericMethod(T t) {}
}