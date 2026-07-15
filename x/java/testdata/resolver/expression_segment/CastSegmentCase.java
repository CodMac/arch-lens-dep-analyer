package com.example.resolver.segmenter.cast;

import java.util.List;
import java.util.Map;

public class CastSegmentCase {
    private Object obj;

    public void testAllCastSegments() {
        // Case 1: 基础向下转型
        String s1 = (String) obj; // 行 11

        // Case 2: 带泛型的集合转型
        List<String> s2 = (List<String>) obj; // 行 14

        // Case 3: 全限定类名转型
        Map s3 = (java.util.Map) obj; // 行 17

        // Case 4: 多重强转 (Double Cast)
        Runnable r = (Runnable)(Object) obj; // 行 20

        // Case 5: 强转后立即进行方法调用
        ((Sub) obj).perform(); // 行 23

        // Case 6: 强转后立即访问属性字段
        String status = ((Dummy) obj).status; // 行 26

        // Case 7: 传统 instanceof 检查
        if (obj instanceof String) {} // 行 29

        // Case 8: 模式匹配 instanceof (Java 14+)
        if (obj instanceof String str) {} // 行 32
    }

    public static class Sub {
        public void perform() {}
    }
    public static class Dummy {
        public String status;
    }
}