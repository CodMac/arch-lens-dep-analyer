package com.example.display;

/**
 * 枚举：常量带类体（匿名子类形态）+ 枚举内嵌套类。
 * 已知边界：常量类体内的字段/方法当前挂到枚举本身（EnclosingQN=枚举，TypeQN=枚举）。
 */
public enum DisplayEnumBody {

    ADD {
        int addBodyField;

        @Override
        int apply(int a, int b) {
            return a + b;
        }
    },

    SUB {
        @Override
        int apply(int a, int b) {
            return a - b;
        }
    };

    abstract int apply(int a, int b);

    static class Meta {
        int metaField;
    }
}
