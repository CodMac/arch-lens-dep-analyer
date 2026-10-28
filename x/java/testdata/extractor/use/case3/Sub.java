package com.example.rel.use.case3.pk2;

import com.example.rel.use.case3.pk1.Base;

public class Sub extends Base {
    public void check() {
        // Source: com.example.rel.use.case3.pk2.Sub.check() (METHOD)
        // Target: com.example.rel.use.case3.pk1.Base.protectedVar (FIELD)
        System.out.println(protectedVar); // [Case 5] 应解析成功 (Protected 可见)
        System.out.println(packageVar);   // [Case 6] 应解析失败或标记为 External (跨包不可见)
    }
}
