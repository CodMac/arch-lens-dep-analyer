package com.example.rel.use.case5;

public class ClosureTest {
    private String context = "Outer";

    public void run() {
        Runnable r = new Runnable() {
            private String context = "Inner";
            @Override
            public void run() {
                // Source: com.example.rel.use.case5.ClosureTest.run().anonymousClass$1.run() (METHOD)
                // Target: com.example.rel.use.case5.ClosureTest.run().anonymousClass$1.context (FIELD)
                System.out.println(context); // [Case 9] 应解析为匿名内部类的 Field
            }
        };

        List<String> list = List.of("A1", "A2");
        list.forEach(item -> {
            // Source: com.example.rel.use.case5.ClosureTest.run().lambda$1 (LAMBDA)
            // Target: com.example.rel.use.case5.ClosureTest.context (FIELD)
            System.out.println(context); // [Case 10] 应解析为外部类的 Field (Lambda 闭包)
        });
    }
}
