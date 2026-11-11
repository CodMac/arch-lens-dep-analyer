package com.example.display;

/**
 * 注解：元素方法 + 嵌套注解 + 嵌套枚举。
 * 嵌套类型的 TypeQN 应为外层注解。
 */
public @interface DisplayAnnotation {

    String value();

    int count() default 0;

    @interface InnerAnno {
        int n();
    }

    enum Level {
        LOW, HIGH
    }
}
