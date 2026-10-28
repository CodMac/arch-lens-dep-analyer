package com.example.base.test;

// 目标：验证 docs/qn-format.md §4.2 条件序号（$N 重名消歧）盲区。
// Java 中类型名与变量名分属不同命名空间，同一类内嵌套类与同名字段合法，
// 应触发 applyUniqueQN 的 occurrences > 1 分支：首个不带 $N，从第 2 个开始 $2、$3...
public class DupNameTest {
    // 嵌套类 Inner（首个出现，不带 $N）
    class Inner {
    }

    // 字段与嵌套类同名（第 2 个出现）→ 应消歧为 Inner$2
    int Inner = 0;

    // 静态嵌套类 Config（首个出现）
    static class Config {
    }

    // 静态字段与静态嵌套类同名 → 应消歧为 Config$2
    static String Config = "x";

    // 普通字段不重名，保持原名
    int count = 1;

    // 方法内的局部类与局部变量分属类型/变量命名空间，同一方法内可重名
    public void test() {
        class Local {
        }
        Local ref = new Local(); // 变量名 ref 与类型名 Local 不同，不冲突

        // 两个同名局部变量分属不同 block（block$1/block$2），各自容器独立计数，不触发 $N
        {
            int dup = 1;
        }
        {
            int dup = 2;
        }
    }
}
