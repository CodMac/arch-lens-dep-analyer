package com.example.lombok;

import lombok.Setter;

@Setter
public class SetterOnly {
    private String name;
    private int age;
    private boolean active;
}

public class SetterUsageTest {
    public void testSetters() {
        SetterOnly obj = new SetterOnly();

        // 应该能识别所有setter方法
        obj.setName("Test");
        obj.setAge(25);
        obj.setActive(true);

        // 不应该有getter方法（除了默认构造器）
        // String name = obj.getName(); // 这个方法不存在
    }
}