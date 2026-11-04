package com.example.rel;

// C-7 限定 this 表达式修复验证：X.this.xxx 应解析为 X 的实例成员调用，
// 而非虚增 X() 构造器 CALL + 丢失真实方法 CALL。
// 典型场景：匿名类/内部类内调用外层类的实例成员。
public class QualifiedThisSuite {

    public static void main(String[] args) {
        // 1. 匿名类内限定 this 调用外层实例方法（CALL）
        Runnable r = new Runnable() {
            @Override
            public void run() {
                // Source: com.example.rel.QualifiedThisSuite$1.run() (METHOD)
                // Target: com.example.rel.QualifiedThisSuite.helper() (METHOD)
                // Mores: {
                //   "java.rel.ast_kind": "method_invocation",
                //   "java.rel.call.receiver": "this",
                //   "java.rel.raw_text": "QualifiedThisSuite.this.helper()"
                // }
                QualifiedThisSuite.this.helper();
            }
        };
        r.run();

        // 2. 内部类内限定 this 调用外层实例字段（USE）
        Inner inner = new Inner();
        inner.run();

        // 3. 匿名类内限定 this 调用外层实例字段（USE）
        Runnable r2 = new Runnable() {
            @Override
            public void run() {
                // Source: com.example.rel.QualifiedThisSuite$2.run() (METHOD)
                // Target: com.example.rel.QualifiedThisSuite.value (FIELD)
                // Mores: {
                //   "java.rel.ast_kind": "field_access",
                //   "java.rel.raw_text": "QualifiedThisSuite.this.value"
                // }
                int v = QualifiedThisSuite.this.value;
            }
        };
        r2.run();
    }

    class Inner {
        void run() {
            // Source: com.example.rel.QualifiedThisSuite.Inner.run() (METHOD)
            // Target: com.example.rel.QualifiedThisSuite.helper() (METHOD)
            // Mores: {
            //   "java.rel.ast_kind": "method_invocation",
            //   "java.rel.call.receiver": "this",
            //   "java.rel.raw_text": "QualifiedThisSuite.this.helper()"
            // }
            QualifiedThisSuite.this.helper();
        }
    }

    public int value = 42;

    public void helper() {}
}
