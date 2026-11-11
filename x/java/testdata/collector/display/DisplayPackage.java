package com.example.display;

/**
 * 展示层字段主样例：包 → 顶层类 → 嵌套类 → 内部类方法 → 局部类 → 匿名类 → lambda → 作用域块。
 * 用于验证 NamespaceQN / EnclosingQN / TypeQN 的完整 5 层链。
 */
public class DisplayPackage {

    private int topField;

    /** 嵌套静态类 */
    public static class Nested {
        int nestedField;

        void nestedMethod() {
            int nestedLocal = 1;
        }
    }

    /** 内部类（非静态） */
    public class Inner {
        int innerField;

        void innerMethod() {
            int innerLocal = 2;
        }
    }

    /** 顶层类方法：局部类 + 匿名类 + lambda + 块 */
    public void topMethod() {
        int methodLocal = 3;

        // 局部类
        class Local {
            int localField;

            void localMethod() {
                int localMethodLocal = 4;
            }
        }
        Local l = new Local();

        // 匿名类
        Runnable anon = new Runnable() {
            @Override
            public void run() {
                int anonLocal = 5;
            }
        };
        anon.run();

        // lambda
        Runnable lambda = () -> {
            int lambdaLocal = 6;
        };
        lambda.run();

        // 作用域块
        {
            int blockLocal = 7;
        }
    }
}
