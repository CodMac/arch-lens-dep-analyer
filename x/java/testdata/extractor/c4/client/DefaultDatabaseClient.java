package com.example.rel.c4.client;

import io.r2dbc.spi.Parameter;
import io.r2dbc.spi.Parameters;

import java.util.List;

/**
 * DefaultDatabaseClient 验证 C-4 单类型导入遮蔽同包类型（模拟 spring-r2dbc
 * DefaultDatabaseClient 真实形态，缺陷 #85）：
 *
 *   import io.r2dbc.spi.Parameter;   // 单类型导入外部库
 *   package com.example.rel.c4.client; // 注意：与 core.Parameter 不同包
 *
 * 场景：本类引用 Parameter 时，按 JLS §7.5.1 单类型导入应解析为
 * io.r2dbc.spi.Parameter（外部，import 包前缀），而非同包/同名其他类型。
 *
 * 修复前：PreciseResolve 第 2 步 import 全局查 QN miss（io.r2dbc.spi 不在
 * GlobalContext），回退第 3 步同包查找。因本类与 core.Parameter 不同包，
 * 同包查找 miss，继续回退到第 4 步裸全局查找——若 GlobalContext 中存在
 * 任意 Parameter（如 core.Parameter），会错误命中。
 * 修复后：单类型导入存在但全局 miss -> 直接外部兜底 io.r2dbc.spi.Parameter。
 */
class DefaultDatabaseClient {

    Parameter from(Object value) {
        return Parameter.from(value);
    }

    List<Parameter> params() {
        return java.util.Collections.emptyList();
    }

    void use(Parameter p) {
        Parameters.in(p);
    }
}
