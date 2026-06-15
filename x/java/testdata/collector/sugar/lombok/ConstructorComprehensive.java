package com.example.lombok;

import lombok.NoArgsConstructor;
import lombok.AllArgsConstructor;
import lombok.RequiredArgsConstructor;

// @NoArgsConstructor: 无参构造器
@NoArgsConstructor
@AllArgsConstructor
public class ConstructorComprehensive {
    private String name;
    private int age;

    @RequiredArgsConstructor
    public static class RequiredArgsConstructor {
        private final String requiredField;  // final字段 - @RequiredArgsConstructor会生成构造器
        private String optionalField;

        @RequiredArgsConstructor
        public static class InnerClass {
            private final int id;
            private String description;
        }
    }
}

public class ConstructorUsageTest {
    public void testConstructors() {
        // 无参构造器
        ConstructorComprehensive obj1 = new ConstructorComprehensive();

        // 全参构造器
        ConstructorComprehensive obj2 = new ConstructorComprehensive("Alice", 25);

        // @RequiredArgsConstructor生成的构造器
        ConstructorComprehensive.RequiredArgsConstructor reqObj =
            new ConstructorComprehensive.RequiredArgsConstructor("required");

        // 内部类的构造器
        ConstructorComprehensive.RequiredArgsConstructor.InnerClass inner =
            new ConstructorComprehensive.RequiredArgsConstructor.InnerClass(123);
    }
}