package com.example.rel.c4.core;

/**
 * Parameter 同包废弃类（模拟 org.springframework.r2dbc.core.Parameter）。
 *
 * 本文件【无】io.r2dbc.spi.Parameter 单类型导入，引用 Parameter 时应命中自身
 * （同包类型，走 PreciseResolve 第 3 步包内查找）。
 */
class Parameter {

    static Parameter from(Object value) {
        return new Parameter(value, null);
    }

    Parameter(Object value, Class<?> type) {
    }
}
