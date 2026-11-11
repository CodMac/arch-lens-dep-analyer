package com.example.display;

/**
 * 嵌套匿名与局部类：
 * 匿名类套匿名类、匿名类内局部类、lambda 内局部类、局部类套局部类。
 * 所有内部的 TypeQN 应跳过匿名/lambda/局部层，取最外层具名类。
 */
public class DisplayNestedAnon {

    /** 匿名类套匿名类 */
    void nestedAnon() {
        Runnable outer = new Runnable() {
            @Override
            public void run() {
                Runnable inner = new Runnable() {
                    @Override
                    public void run() {
                    }
                };
                inner.run();
            }
        };
        outer.run();
    }

    /** 匿名类内局部类 */
    void anonWithLocal() {
        Runnable r = new Runnable() {
            @Override
            public void run() {
                class InAnon {
                    int inAnonField;
                }
                new InAnon();
            }
        };
        r.run();
    }

    /** lambda 内局部类 */
    void lambdaWithLocal() {
        Runnable r = () -> {
            class InLambda {
                int inLambdaField;
            }
            new InLambda();
        };
        r.run();
    }

    /** 局部类套局部类 */
    void localNest() {
        class OuterLocal {
            void f() {
                class InnerLocal {
                    int innerLocalField;
                }
                new InnerLocal();
            }
        }
        new OuterLocal().f();
    }
}
