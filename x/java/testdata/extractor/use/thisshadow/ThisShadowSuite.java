package com.example.rel;

// 字段遮蔽回归用例：方法参数与字段同名时，this.field 的 field_access 必须解析到字段，
// 而非被作用域链上的同名参数遮蔽。RHS 裸标识符（无 this 前缀）才解析到参数。
public class ThisShadowSuite {
    private String name = "field";
    private int bean = 1;

    // 构造器：this.name = name
    // ASSIGN
    //   Source: com.example.rel.ThisShadowSuite.ThisShadowSuite(String)
    //   Target: com.example.rel.ThisShadowSuite.name (字段，非参数)
    //   Mores: { "java.rel.raw_text": "this.name = name" }
    // USE (field_access, LHS this.name)
    //   Target: com.example.rel.ThisShadowSuite.name (字段)
    //   Mores: { "java.rel.context": "field_access", "java.rel.raw_text": "this.name", "java.rel.use.receiver": "this" }
    // USE (assignment_expression, RHS 裸 name)
    //   Target: com.example.rel.ThisShadowSuite.ThisShadowSuite(String).name (参数)
    //   Mores: { "java.rel.context": "assignment_expression", "java.rel.raw_text": "this.name = name" }
    public ThisShadowSuite(String name) {
        this.name = name;
    }

    // setName(String name)：this.name = name，语义同上
    public void setName(String name) {
        this.name = name;
    }

    // setBean(int bean)：this.bean = bean，语义同上（int 类型字段/参数同名）
    public void setBean(int bean) {
        this.bean = bean;
    }

    // 无同名参数：this.name 直接解析到字段
    // USE (field_access, return_statement)
    //   Target: com.example.rel.ThisShadowSuite.name (字段)
    public String getShadow() {
        return this.name;
    }

    // 无同名参数：this.name 直接解析到字段
    // USE (field_access)
    //   Target: com.example.rel.ThisShadowSuite.name (字段)
    public void useField() {
        System.out.println(this.name);
    }
}
