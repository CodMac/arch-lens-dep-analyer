package com.test;

import com.test.case1.ParentClass;
import com.test.case1.User;
import com.test.case1.Role;

public class UseCase1 extends ParentClass {
    private int count = 10;

    public void execute(String param, Object obj) {
        // 维度 1: 变量与形参读取 (Local Variable & Parameter Use)

        // Line 16: 场景 1.1 - 局部变量读取 (读取 x)
        int x = 100;
        int y = x + 1;

        // Line 20: 场景 1.2 - 方法形参读取 (读取 param)
        String trimmed = param.trim();

        // Line 23: 场景 1.3 - Catch 异常变量读取 (读取 e)
        try {
            System.out.println("try");
        } catch (Exception e) {
            e.printStackTrace();
        }


        // 维度 2: 成员字段与继承属性读取 (Field & Parent Access)

        // Line 33: 场景 2.1 - 当前类实例字段读取 (读取 count)
        int currentCount = this.count;

        // Line 36: 场景 2.2 - 父类继承字段读取 (读取 parentField)
        String parentVal = super.parentField;

        // Line 39: 场景 2.3 - 对象属性直接读取 (读取 user.name)
        User user = new User();
        String userName = user.name;


        // 维度 3: 常量与枚举读取 (Constants & Enums)

        // Line 46: 场景 3.1 - 类静态常量读取 (读取 Constants.DEFAULT_NAME)
        String defaultName = Constants.DEFAULT_NAME;

        // Line 49: 场景 3.2 - 枚举项读取 (读取 Role.ADMIN)
        Role currentRole = Role.ADMIN;


        // 维度 4: 高级语法读取与未导入保底

        // Line 55: 场景 4.1 - 模式匹配变量读取 (读取 s)
        if (obj instanceof String s) {
            int len = s.length();
        }

        // Line 60: 场景 4.2 - 未导入外部类的静态常量读取 (保底)
        int timeout = UnimportedConfig.TIMEOUT;
    }

    public static class Constants {
        public static final String DEFAULT_NAME = "arch-lens";
    }
}