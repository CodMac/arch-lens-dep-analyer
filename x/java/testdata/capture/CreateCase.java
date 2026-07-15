package com.example.capture;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class CreateCase {
    public void testBasicCreation() {
        // 场景 1: 普通无参对象创建
        Object obj = new Object(); // Line 11

        // 场景 2: 带参对象创建
        String str = new String("Hello"); // Line 14

        // 场景 3: 匿名内部类创建
        Runnable r = new Runnable() { // Line 17
            @Override
            public void run() {}
        };
    }

    public void testNestedCreation() {
        // 场景 4: 嵌套限定类型创建 (普通内部类)
        Outer.Inner inner = new Outer.Inner(); // Line 25

        // 场景 5: 深层嵌套限定类型创建
        A.B.C c = new A.B.C(); // Line 28
    }

    public void testGenericCreation() {
        // 场景 6: 菱形语法 (Diamond operator) 创建
        List<String> list = new ArrayList<>(); // Line 33

        // 场景 7: 显式指定泛型类型的创建
        Map<String, Integer> map = new HashMap<String, Integer>(); // Line 36

        // 场景 8: 嵌套泛型类型的创建
        List<List<String>> complexList = new ArrayList<List<String>>(); // Line 39
    }

    public void testArrayCreation() {
        // 场景 9: 基本类型数组创建 (带长度)
        int[] numbers = new int[10]; // Line 44

        // 场景 10: 基本类型数组创建 (带初始值)
        int[] predefined = new int[]{1, 2, 3}; // Line 47

        // 场景 11: 对象类型数组创建
        String[] strings = new String[5]; // Line 50

        // 场景 12: 多维数组创建
        int[][] matrix = new int[3][4]; // Line 53
    }

    public void testChainedCreation() {
        // 场景 13: 创建后立即进行方法调用 (链式调用)
        int length = new String("test").length(); // Line 58

        // 场景 14: 创建后立即访问字段
        int val = new DummyClass().publicField; // Line 61
    }

    public void testCreationInExpressions() {
        // 场景 15: 在方法传参中创建对象
        doSomething(new DummyClass()); // Line 66

        // 场景 16: 在 Return 语句中创建对象
        return new DummyClass(); // Line 69
    }

    // 辅助类用于模拟外部依赖
    static class Outer { static class Inner {} }
    static class A { static class B { static class C {} } }
    static class DummyClass { public int publicField = 10; }
    private void doSomething(Object o) {}
}