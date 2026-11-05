package com.example.rel.c6;

// C-6 重载决议实参类型推断用例。
// 复现 spring AnnotationsScanner 的 hasPlainJavaAnnotationsOnly(Object)/(Class<?>) 双重重载：
// 修复前 inferArgType 对 identifier/field_access/method_invocation 实参返回 unknown，
// pickBestOverload unknown 只 +10，(Object)/(Class) 并列取 entries[0]（Object 版），(Class) 重载 0 命中。

class TypeSource {
    static Class<?> clazz() { return null; }
    static String str() { return null; }
}

class Holder {
    Class<?> klass = null;
    String text = null;
}

class OverloadProbe {
    Holder holder = new Holder();
    static Class<?> staticKlass = null;

    // 双重重载：Object 版在前（复现 spring entries[0]=Object 版）。
    static boolean accept(Object o) { return true; }
    static boolean accept(Class<?> c) { return true; }

    void probe(Class<?> source, String text) {
        boolean r1 = accept(text);      // identifier -> String -> Object 版
        boolean r2 = accept(source);    // identifier -> Class -> Class 版
        boolean r3 = accept(staticKlass); // identifier(静态字段) -> Class -> Class 版
        boolean r4 = accept(holder.klass); // field_access -> Class -> Class 版
        boolean r5 = accept(TypeSource.clazz()); // method_invocation -> Class -> Class 版
        boolean r6 = accept(TypeSource.str());   // method_invocation -> String -> Object 版
        boolean r7 = accept(new Object());       // object_creation -> Object 版
        boolean r8 = accept("literal");          // string_literal -> String -> Object 版
    }
}
