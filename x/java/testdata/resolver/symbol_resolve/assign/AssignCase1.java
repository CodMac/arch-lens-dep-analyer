package com.test;

import com.test.case1.ParentClass;
import com.test.case1.User;

public class AssignCase1 extends ParentClass {
    private int count = 0;
    private int total = 100;

    public void execute(User user) {
        // 维度 1: 变量声明初始化与重新赋值 (Local Variable Assign)

        // Line 15: 场景 1.1 - 局部变量声明并初始化 (写入 x)
        int x = 10;

        // Line 18: 场景 1.2 - 已有局部变量重新赋值 (写入 x)
        x = 20;


        // 维度 2: 成员字段与继承属性写入 (Field & Property Assign)

        // Line 24: 场景 2.1 - 当前类实例字段写入 (写入 count)
        this.count = 100;

        // Line 27: 场景 2.2 - 父类继承字段写入 (写入 parentField)
        super.parentField = "new_value";

        // Line 30: 场景 2.3 - 跨类对象属性写入 (写入 user.name)
        user.name = "Bob";


        // 维度 3: 复合赋值与自增自减 (Compound & Update Assign)

        // Line 36: 场景 3.1 - 复合运算符赋值 (写入 total)
        this.total += 50;

        // Line 39: 场景 3.2 - 自增运算符 (写入 count)
        this.count++;


        // 维度 4: 类静态常量/字段与未导入保底

        // Line 45: 场景 4.1 - 静态成员字段写入 (写入 Config.DEBUG_MODE)
        Config.DEBUG_MODE = true;

        // Line 48: 场景 4.2 - 未导入外部类的静态字段写入 (保底)
        UnimportedConfig.FLAG = false;
    }

    public static class Config {
        public static boolean DEBUG_MODE = false;
    }
}