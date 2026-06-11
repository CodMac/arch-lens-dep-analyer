package com.example.lombok;

import lombok.Data;

@Data
public class DataComprehensive {
    private String name;
    private int age;
    private boolean active;
    private String email;
}

public class DataUsageTest {
    public void testDataMethods() {
        DataComprehensive user = new DataComprehensive();

        // Setters
        user.setName("Alice");
        user.setAge(25);
        user.setActive(true);
        user.setEmail("alice@example.com");

        // Getters
        String name = user.getName();
        int age = user.getAge();
        boolean active = user.isActive();
        String email = user.getEmail();

        // Equals and hashCode (implicitly used in collections)
        DataComprehensive another = new DataComprehensive();
        another.setName("Alice");
        another.setAge(25);
        another.setActive(true);
        another.setEmail("alice@example.com");

        boolean equals = user.equals(another);
        int hashCode = user.hashCode();

        // toString
        String str = user.toString();
    }
}