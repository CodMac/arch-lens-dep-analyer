package com.example.rel.use.case2;

public class Child extends Parent {
    private int count = 20; // 遮蔽父类字段

    public void print() {
        // Source: com.example.rel.use.case2.Child.print() (METHOD)
        // Target: com.example.rel.use.case2.Child.count (FIELD)
        System.out.println(count); // [Case 3] 应解析为本类的 Field (count=20)
        // Source: com.example.rel.use.case2.Child.print() (METHOD)
        // Target: com.example.rel.use.case2.Parent.TAG (FIELD)
        System.out.println(TAG);   // [Case 4] 应跨类解析为 Parent 的 Static Field
    }
}
