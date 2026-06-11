package com.example.lombok;

import lombok.Getter;

@Getter
public class GetterOnly {
    private String name;
    private final int age;  // final字段也可以有getter
    private boolean active;
}

public class GetterUsageTest {
    public void testGetters() {
        GetterOnly obj = new GetterOnly();

        // 应该能识别所有getter方法
        String name = obj.getName();
        int age = obj.getAge();
        boolean active = obj.isActive();

        // 不应该有setter方法
        // obj.setName("Test"); // 这个方法不存在
    }
}