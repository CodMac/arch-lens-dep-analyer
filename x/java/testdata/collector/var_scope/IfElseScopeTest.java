package com.example.base.test;

// 目标：验证 if/else 块内变量的作用域重定位（Bug A 回归）。
// tree-sitter 中 if/else 是同一个 if_statement 下的两个 block 子节点
// （else 无独立节点），旧逻辑固定匹配第一个 block 导致 else 块变量被误挂到 if 体块。
// 修复后：if 体变量 -> block$1，else 块变量 -> block$2。
public class IfElseScopeTest {
    public void test(int flag) {
        if (flag > 0) {
            int ifVar = 1;
            System.out.println(ifVar);
        } else {
            int elseVar = 2;
            System.out.println(elseVar);
        }
    }

    // 增强 for 循环体内声明的变量须归 for 体块，不得错挂外层 if 体块（Bug A 延伸）。
    // findNearestBlockParent 旧版缺 enhanced_for_statement 分支，向上命中 if_statement
    // 后遍历其 block 子节点，for 体内变量被误挂到外层 if 体块。
    public void forEach(int flag) {
        if (flag > 0) {
            for (String item : new String[]{"a"}) {
                int loopVar = 1;
                System.out.println(loopVar);
            }
        }
    }
}
