package com.example.resolver.segmenter.create;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;

public class CreateSegmentCase {

    private Object obj;

    public void testAllCreateSegments() {
        // Case 1: 基础对象创建
        User user = new User(); // 行 13

        // Case 2: 泛型对象创建
        List<String> list = new ArrayList<String>(); // 行 16

        // Case 3: 内部类创建
        Outer.Inner inner = new Outer.Inner(); // 行 19

        // Case 4: 多层深嵌套内部类
        A.B.C cNode = new A.B.C(); // 行 22

        // Case 5: 基础数据类型数组
        int[] arr = new int[10]; // 行 25

        // Case 6: 创建后立即链式调用方法
        new Outer.Inner().perform(); // 行 28

        // Case 7: 创建后立即访问内部字段
        String s = new Dummy().status; // 行 31

        // Case 8: 创建后立即进行数组索引访问
        String[] strs = new String[5][0]; // 行 34
    }

    public static class User {}
    public static class Outer {
        public static class Inner {
            public void perform() {}
        }
    }
    public static class A {
        public static class B {
            public static class C {}
        }
    }
    public static class Dummy {
        public String status;
    }
}