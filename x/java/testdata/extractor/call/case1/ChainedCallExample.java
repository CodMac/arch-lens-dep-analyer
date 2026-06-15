package com.example.chained;

public class ChainedCallExample {

    public static class Builder {
        private String name;
        private int age;

        public Builder name(String name) {
            this.name = name;
            return this;
        }

        public Builder age(int age) {
            this.age = age;
            return this;
        }

        public ChainedCallExample build() {
            return new ChainedCallExample(this);
        }
    }

    public ChainedCallExample(Builder builder) {
        this.name = builder.name;
        this.age = builder.age;
    }

    private String name;
    private int age;

    public String getName() {
        return name;
    }

    public static void testChainedCalls() {
        // 链式调用1：使用Builder模式
        ChainedCallExample obj1 = new Builder()
                .name("Alice")
                .age(25)
                .build();

        // 链式调用2：普通链式调用
        String result = obj1.getName().toUpperCase();

        // 链式调用3：深层链式调用
        String chained = obj1.getName().toUpperCase().trim();
    }
}