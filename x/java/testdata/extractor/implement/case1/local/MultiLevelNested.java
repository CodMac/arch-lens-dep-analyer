package com.example.rel.implement.case1.local;

/**
 * MultiLevelNested 验证多级嵌套名（A.B.C 形态，C-10 修复变体）：
 *   implements MultiLevelNested.Level1.Level2
 *
 * 同文件 self 限定，包内拼 QN 路径即可命中多级嵌套。
 */
class MultiLevelNested implements MultiLevelNested.Level1.Level2 {

    interface Level1 {
        interface Level2 {
            String name();
        }
    }

    @Override
    public String name() {
        return "multi";
    }
}
