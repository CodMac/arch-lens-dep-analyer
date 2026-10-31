package com.example.rel.use.case9;

// 目标：验证跨方法同名变量的 USE 不串扰（Bug B 回归）。
// 两个方法各自声明同名局部变量 info，且另一个方法在 else 块中使用 info。
// Bug B 根因：ResolveVar 失败后，resolveIdent 的类型名兜底走 PreciseResolve->
// FindByShortName 无条件取 entries[0]，跨方法命中另一方法的同名字段/变量。
// 修复后：ResolveVar 成功命中本方法作用域变量，USE 目标必须是本方法/本块变量；
// 作用域外引用（methodC 的 info 未声明）不得解析到任何方法内的 info。
public class CrossMethodVarSuite {
    // L8-12：方法 A 声明并返回 info
    public Object methodA(int flag) {
        String info = "A";
        // Source: com.example.rel.use.case9.CrossMethodVarSuite.methodA(int)
        // Target: com.example.rel.use.case9.CrossMethodVarSuite.methodA(int).info
        return info;
    }

    // L14-23：方法 B 在 else 块内使用 info（须解析到方法 B 的 else 块变量）
    public Object methodB(int flag) {
        if (flag > 0) {
            System.out.println("positive");
        } else {
            String info = "B";
            // Source: com.example.rel.use.case9.CrossMethodVarSuite.methodB(int)
            // Target: com.example.rel.use.case9.CrossMethodVarSuite.methodB(int).block$2.info
            // 注意：不得解析到 methodA(int).info（跨方法同名串扰，Bug B）
            return info;
        }
        return null;
    }

    // L26-29：方法 C 引用作用域外的 info（未声明）。ResolveVar 失败后，
    // 类型兜底不得命中 methodA/methodB 的 info（跨方法串扰），应外部兜底。
    public Object methodC() {
        return info;
    }
}
