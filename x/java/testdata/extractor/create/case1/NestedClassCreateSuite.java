package com.example.rel.create.case1;

/**
 * NestedClassCreateSuite 验证带限定符 new 表达式的 CREATE/CALL 提取（BUG-2 回归）。
 *
 * 场景：new Outer.Inner() 的类型是 scoped_type_identifier（跨文件静态嵌套类），
 *  修复前 action query 的 object_creation_expression 只匹配 type_identifier 简单类名
 *  （不含 scoped_type_identifier），导致这类 new 漏报 CREATE 与 CALL。
 *
 * 归属期望（Source 均为方法）：
 *  - L23 new Outer.Inner()                    -> CREATE 目标 com.example.rel.create.case1.Outer.Inner
 *  - L23 new Outer.Inner()                    -> CALL 目标 com.example.rel.create.case1.Outer.Inner.Inner()
 *  - L24 new Outer.Inner("n")                 -> CREATE 目标 com.example.rel.create.case1.Outer.Inner
 *  - L25 new NestedClassCreateSuite.InstanceInner() -> CREATE 目标 ...NestedClassCreateSuite.InstanceInner
 */
public class NestedClassCreateSuite {
    public class InstanceInner {
        public InstanceInner() {
        }
    }

    public void createNested() {
        Outer.Inner a = new Outer.Inner();
        Outer.Inner b = new Outer.Inner("n");
        NestedClassCreateSuite.InstanceInner c = new NestedClassCreateSuite.InstanceInner();
    }
}
