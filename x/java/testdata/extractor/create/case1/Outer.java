package com.example.rel.create.case1;

/**
 * Outer 定义静态嵌套类 Inner，供 NestedClassCreateSuite 跨文件创建。
 * 模拟 InitPartitionForEntityRunner + PartitionDateMigrator 的跨文件嵌套类场景（BUG-2）。
 */
public class Outer {
    public static class Inner {
        public Inner() {
        }

        public Inner(String name) {
        }

        public void doInner() {
        }
    }
}
