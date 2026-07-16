package com.test;

import java.util.ArrayList;
import java.util.HashMap;
import com.factory.Product;

public class CreateCase {

    public static class StaticInner {
        public StaticInner() {}
    }
    public class MemberInner {
        public MemberInner() {}
    }

    public void run() {
        // Line 18 - 场景 1.1: 静态内部类创建
        Outer.StaticInner obj1 = new Outer.StaticInner();

        // Line 22 - 场景 1.2: 成员内部类创建
        Outer outer = new Outer();
        Outer.MemberInner obj2 = outer.new MemberInner();

        // Line 25 - 场景 2.1: 导入外部类创建（带泛型擦除）
        ArrayList<String> list = new ArrayList<>();

        // Line 28 - 场景 2.2: 基础外部类创建 (java.lang.String)
        String str = new String("hello");

        // Line 31 - 场景 2.3: 野指针/未导入创建
        Object obj = new SomeUnknownClass();

        // Line 34 - 场景 3.1: 实例化后立即发起链式调用 (Create 的断言锚定在此)
        String res = new Builder().select("SELECT *").build();

        // Line 37 - 场景 4.1: 跨包公开类创建
        Product p = new Product();

        // Line 40 - 场景 5.1: 接口匿名实现类创建
        Runnable r = new Runnable() {
            @Override public void run() {}
        };

        // Line 45 - 场景 7.1: 数组类型创建
        User[] users = new User[5];
    }
}