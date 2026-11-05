package com.example.rel;

import java.util.function.Supplier;

// 接口常量：隐式 public static final
interface AnnoFilter {
    String PLAIN = "plain";
}

// 外部继承字段源（模拟 TagSupport.pageContext）
class ExternalBase {
    protected JspContext pageContext;
}

class JspContext {
    public Object getAttribute(String s) { return null; }
}

class C8FullProbe extends ExternalBase {

    // Form 1: Type::method 方法引用
    boolean f1(boolean b) {
        java.util.function.BooleanSupplier s = AnnoFilter::matches;
        return s.getAsBoolean();
    }
    static boolean matches() { return true; }

    // Form 2: Type.FIELD.method() 接口常量
    int f2() {
        return AnnoFilter.PLAIN.length();
    }

    // Form 3: this.field.method() 外部继承字段
    Object f3() {
        return this.pageContext.getAttribute("x");
    }
}
