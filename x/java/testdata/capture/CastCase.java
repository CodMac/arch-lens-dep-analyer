package com.example.capture;

import java.util.List;
import java.util.Map;

public class CastCase {
    private Object obj;
    private Object input;
    private int intVal;

    public void testAllCastAndInstanceof() {
        // ==================== 基础与经典强转场景 ====================
        // 场景1: 基础向下转型
        String s = (String) obj;

        // 场景2: 基础数据类型转换
        double d = (double) intVal;

        // 场景3: 带泛型的集合转型
        List<String> list = (List<String>) obj;

        // 场景4: 全限定类名转型
        Object x = (java.util.Map) input;

        // ==================== 复杂强转场景 (链式/多重) ====================
        // 场景5: 多重强转 (Double Cast) 并伴随链式调用
        ((Runnable)(Object)input).run();

        // 场景6: 链式调用中的转型
        ((SubClass) obj).targetMethod();

        // ==================== 类型检查与模式匹配 ====================
        // 场景7: 传统 instanceof 检查
        if (obj instanceof String) {
            // do something
        }

        // 场景8: Java 14+ 模式匹配 instanceof
        if (obj instanceof String str) {
            System.out.println(str);
        }
    }

    private static class SubClass {
        void targetMethod() {}
    }
}