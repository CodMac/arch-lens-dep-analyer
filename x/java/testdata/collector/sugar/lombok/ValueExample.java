package com.example.lombok;

import lombok.Value;

@Value
public class ValueExample {
    private String name;
    private int age;
    private boolean active;
}

// @Value 等价于 @Getter @FieldDefaults(makeFinal=true, level=PRIVATE) @AllArgsConstructor @ToString @EqualsAndHashCode
public class ValueUsageTest {
    public void testValueImmutability() {
        // 使用全参构造器（@Value生成的）
        ValueExample user = new ValueExample("Alice", 25, true);

        // 可以使用getter
        String name = user.getName();
        int age = user.getAge();
        boolean active = user.isActive();

        // 不应该有setter（因为是immutable的）
        // user.setName("Bob"); // 编译错误

        // equals和hashCode可用
        ValueExample another = new ValueExample("Alice", 25, true);
        boolean equals = user.equals(another);
        int hashCode = user.hashCode();

        // toString可用
        String str = user.toString();
    }
}