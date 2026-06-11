package com.example.lombok;

import lombok.ToString;

@ToString
public class ToStringExample {
    private String name;
    private int age;
    private boolean active;

    // 可以排除某些字段
    @ToString.Exclude
    private String secret;
}

@ToString(includeFieldNames = true)
public class ToStringDetailedExample {
    private String field1;
    private String field2;
}

public class ToStringUsageTest {
    public void testToString() {
        ToStringExample obj1 = new ToStringExample();
        if (obj1.age > 18) {
            // toString方法应该被识别
            String str = obj1.toString();
        }

        ToStringDetailedExample obj2 = new ToStringDetailedExample();
        String detailedStr = obj2.toString();
    }
}