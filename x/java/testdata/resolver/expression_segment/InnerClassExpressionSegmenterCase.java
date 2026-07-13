package org;

public class InnerClassExpressionSegmenterCase {
    private String outerField = "outer";

    // 静态内部类
    public static class StaticInner {
        public String staticInnerField = "static_inner";
        public void doSomething() {}
        public static void staticDoSomething() {}
    }

    // 普通非静态内部类
    public class NonStaticInner {
        public String innerField = "inner";

        public void testOuterAccess() {
            // 场景 1: 通过 Outer.this 访问外部类字段
            String val1 = InnerClassExpressionSegmenterCase.this.outerField; // Line 19

            // 场景 2: 通过 Outer.this 调用外部类方法
            InnerClassExpressionSegmenterCase.this.toString(); // Line 22
        }
    }

    public void executeContext() {
        // 场景 3: 实例化静态内部类并调用方法
        StaticInner staticObj = new StaticInner();
        staticObj.doSomething(); // Line 29

        // 场景 4: 连续访问内部类的字段
        String val2 = staticObj.staticInnerField; // Line 33

        // 场景 5: 匿名内部类内的链式调用
        Runnable r = new Runnable() {
            @Override
            public void run() {
                // 模拟匿名内部类中访问外部作用域的变量
                staticObj.doSomething(); // Line 39
            }
        };

        // 场景 6: 静态内部类构造函数路径限制
        StaticInner staticObj2 = new InnerClassExpressionSegmenterCase.StaticInner(); // Line 44
        InnerClassExpressionSegmenterCase obj1 = new InnerClassExpressionSegmenterCase(); // Line 45
        InnerClassExpressionSegmenterCase obj2 = new InnerClassExpressionSegmenterCase<String>(); // Line 46 ,这个泛型声明再代码中不存在,这里仅为了测试AST

        // 场景 7: 静态内部类的静态方法多层级调用
        InnerClassExpressionSegmenterCase.StaticInner.staticDoSomething(); // Line 49
    }
}