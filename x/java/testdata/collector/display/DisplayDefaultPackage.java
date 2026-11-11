// 默认包：无 package 声明。NamespaceQN 应为空，其余字段仍正确。
public class DisplayDefaultPackage {

    int field;

    void method() {
        int local = 0;
    }

    static class Nested {
        int nestedField;
    }
}
