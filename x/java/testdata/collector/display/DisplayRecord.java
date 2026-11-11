package com.example.display;

/**
 * record：组件字段 / 紧凑构造器 / 嵌套类 / 显式方法。
 * 组件字段与紧凑构造器块的 TypeQN 应为 record 自身。
 */
public record DisplayRecord(int x, int y) {

    // 紧凑构造器
    public DisplayRecord {
        if (x < 0) {
            throw new IllegalArgumentException("x < 0");
        }
    }

    static class Helper {
        int helperField;
    }

    int sum() {
        return x + y;
    }
}
