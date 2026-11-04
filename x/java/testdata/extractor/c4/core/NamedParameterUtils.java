package com.example.rel.c4.core;

import io.r2dbc.spi.Parameter;

import java.util.List;

/**
 * NamedParameterUtils 验证 C-4 同包遮蔽场景（模拟 spring-r2dbc NamedParameterUtils
 * 真实形态，缺陷 #92）：
 *
 *   package com.example.rel.c4.core;      // 与同包废弃类 Parameter 同包
 *   import io.r2dbc.spi.Parameter;        // 单类型导入外部库
 *
 * 按 JLS §7.5.1，单类型导入 io.r2dbc.spi.Parameter 应遮蔽同包 core.Parameter。
 *
 * 修复前：PreciseResolve 第 2 步 import 全局查 QN miss（io.r2dbc.spi 不在
 * GlobalContext），回退第 3 步同包查找拼 QN 命中 core.Parameter——RETURN 与
 * TYPE_ARG 目标全部错归同包废弃类。
 * 修复后：单类型导入存在但全局 miss -> 外部兜底 io.r2dbc.spi.Parameter。
 */
class NamedParameterUtils {

    List<Parameter> expand(Parameter p) {
        return java.util.Collections.singletonList(p);
    }

    Parameter unwrap(Object o) {
        return (Parameter) o;
    }
}
