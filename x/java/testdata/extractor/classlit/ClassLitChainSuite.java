package com.example.rel;

import java.util.List;

public class ClassLitChainSuite {

    // Xxx.class 带后续段：成员应在 java.lang.Class 上解析（.class. receiver 精度修复）
    public String chain() {
        String cn = String.class.getCanonicalName();
        return cn;
    }

    // 纯 Xxx.class（无段）：作为 USE 目标是类型引用本身
    public void bare() {
        Class<?> k = List.class;
    }
}
