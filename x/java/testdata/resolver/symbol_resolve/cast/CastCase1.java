package com.test;

import java.util.List;
import java.util.ArrayList;
import com.test.case1.User;
import com.test.case1.Sub;
import com.test.case1.Dummy;

public class CastCase1 {
    public void execute(Object obj) {
        // 维度 1: 基础与泛型强转 (Basic & Generic Cast)

        // Line 14: 场景 1.1 - 同包/已导入的本地类强转
        User user1 = (User) obj;

        // Line 17: 场景 1.2 - 带泛型的集合类强转（需剥离泛型擦除，还原 java.util.List）
        List<String> list = (List<String>) obj;

        // Line 20: 场景 1.3 - 全限定名强转（直接提取，不冗余解析）
        java.util.Map map = (java.util.Map) obj;


        // 维度 2: 多重强转与嵌套 (Multiple & Nested Cast)

        // Line 26: 场景 2.1 - 多重强转 (Double Cast)，应当精准捕获最外层的目标类型
        Runnable r = (Runnable)(Object) obj;

        // Line 29: 场景 2.2 - 带括号的强转优先级，验证一元化收敛，剥离冗余括号
        User user2 = ((User) obj);


        // 维度 3: 类型检查 (Instanceof Check)

        // Line 35: 场景 3.1 - 传统 instanceof 检查（应当解析出 java.lang.String 依赖）
        if (obj instanceof String) { System.out.println("String"); }

        // Line 38: 场景 3.2 - 模式匹配 instanceof (Java 14+ Pattern Matching)
        if (obj instanceof User u) { System.out.println(u.getName()); }


        // 维度 4: 强转后的链式流转 (Cast with Chained Invocation)

        // Line 44: 场景 4.1 - 强转后立即发起方法调用（Cast 动作捕获点应当对应 ((Sub) obj)，并修正后续类型）
        ((Sub) obj).perform();

        // Line 47: 场景 4.2 - 强转后立即访问属性字段（Cast 动作捕获点对应 ((Dummy) obj)）
        String s = ((Dummy) obj).status;

        // Line 50: 场景 4.3 - 外部未导入类的强转流转（由于本地无 ArrayList 源码，验证保底和 External=true）
        int size = ((ArrayList) obj).size();
    }
}