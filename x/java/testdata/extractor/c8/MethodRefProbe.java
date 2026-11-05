package com.example.rel;

import java.util.function.BooleanSupplier;

interface MergedAnnotation {
    boolean isPresent();
    boolean VALUE();
    boolean withNonMergedAttributes();
    boolean synthesize();
    static MergedAnnotation from() { return null; }
}

public class MethodRefProbe {
    static String FACTORY_BEAN_PREFIX = "&";
    static final String PLAIN = "plain";

    // Type::method —— 接收者是类型名的静态方法引用
    boolean a(boolean b) {
        BooleanSupplier s = MergedAnnotation::isPresent;
        return s.getAsBoolean();
    }

    // Type.FIELD.length() —— 静态字段访问后续方法调用
    int b() {
        return MergedAnnotation.VALUE.length();
    }

    // this.field 外部继承字段的后续方法调用
    int c() {
        return FACTORY_BEAN_PREFIX.length();
    }
}

class NewRefProbe {
    static class Inner {
        Inner() {}
        Inner(int x) {}
    }
    java.util.function.Supplier<Inner> d() {
        return Inner::new;
    }
}
