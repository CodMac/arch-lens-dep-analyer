package com.example.lombok;

import lombok.Getter;
import lombok.Setter;
import lombok.EqualsAndHashCode;

@EqualsAndHashCode
@Getter
@Setter
public class EqualsAndHashCodeExample {
    private String id;
    private String name;
    private int version;
}

public class EqualsAndHashCodeUsageTest {
    public void testEqualsAndHashCode() {
        EqualsAndHashCodeExample obj1 = new EqualsAndHashCodeExample();
        obj1.setId("123");
        obj1.setName("Test");
        obj1.setVersion(1);

        EqualsAndHashCodeExample obj2 = new EqualsAndHashCodeExample();
        obj2.setId("123");
        obj2.setName("Test");
        obj2.setVersion(1);

        // equals方法应该被识别
        boolean equalsResult = obj1.equals(obj2);

        // hashCode方法应该被识别
        int hashCode = obj1.hashCode();

        // canEqual方法（Lombok生成的）
        boolean canEqual = obj1.canEqual(obj2);
    }
}