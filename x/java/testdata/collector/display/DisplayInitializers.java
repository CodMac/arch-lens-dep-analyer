package com.example.display;

import java.util.function.Supplier;

/**
 * 初始化块与字段初始化：
 * 静态初始化块内局部类、实例初始化块内局部类、
 * 字段初始化 lambda / 匿名类 / 方法引用。
 */
public class DisplayInitializers {

    // 字段初始化：lambda
    private final Supplier<String> lambdaField = () -> "x";

    // 字段初始化：匿名类
    private final Runnable anonField = new Runnable() {
        @Override
        public void run() {
        }
    };

    // 字段初始化：方法引用
    private final Supplier<DisplayInitializers> ctorField = DisplayInitializers::new;

    // 静态初始化块内局部类
    static {
        class InStaticBlock {
            int staticBlockField;
        }
        new InStaticBlock();
    }

    // 实例初始化块内局部类
    {
        class InInstanceBlock {
            int instanceBlockField;
        }
        new InInstanceBlock();
    }
}
