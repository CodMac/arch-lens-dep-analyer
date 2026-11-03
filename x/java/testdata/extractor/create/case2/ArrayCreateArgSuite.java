package com.example.rel.create.case2;

import java.util.List;

/**
 * ArrayCreateArgSuite 验证「数组创建作为方法调用实参」的 CREATE/CALL 提取（BUG-B 回归）。
 *
 * 场景：List.of(new String[]{"a", "b"}) —— 数组创建（array_creation_expression）
 * 嵌在方法调用（method_invocation List.of）的实参里。
 * 修复前 create_target 分支对 array_creation_expression 产 CREATE + CALL 两条：
 * CALL 沿 findCoreChain 从数组 type_identifier 向上爬升，误命中外层 method_invocation
 * （List.of），产生一条重复 CALL（raw = new String[]{...}，ast_kind = type_identifier，
 * context = array_creation_expression）。
 * 修复后 array 只产 CREATE（is_array=true），不再误产 CALL。
 *
 * 归属期望（Source 均为 arrayAsArg()）：
 *  - L13 new String[]{"a", "b"}            -> CREATE 目标 String（is_array=true）
 *  - L13 List.of(...)                      -> CALL 目标 java.util.List.of()（真实调用，恰 1 条）
 *  - 负面：不存在从数组产生的 CALL（ast_kind=type_identifier, context=array_creation_expression）
 */
public class ArrayCreateArgSuite {
    public void arrayAsArg() {
        List<String> list = List.of(new String[]{"a", "b"});
    }
}
