package com.example.rel.implement.case1.support;

import com.example.rel.implement.case1.codec.CodecConfigurer;

/**
 * BaseDefaultCodecs 验证【跨包】implements 点分嵌套名（Class.NestedInterface）的
 * IMPLEMENT 提取（C-10 修复核心场景）。
 *
 * 模拟 spring-web BaseDefaultCodecs 的真实形态：本类在 support 子包，
 * implements 的 CodecConfigurer.DefaultCodecs 是 codec 包外层接口的嵌套接口。
 *
 * 修复前：PreciseResolve 对点分嵌套名未拆分外层类解析——
 *   - FindByShortName("CodecConfigurer.DefaultCodecs")：文件内短名表 key 是裸短名
 *     CodecConfigurer，整串必然 miss；
 *   - fc.Imports["CodecConfigurer.DefaultCodecs"]：import 键是外层短名 CodecConfigurer，
 *     整串 miss；
 *   - 包内拼 QN（...support.CodecConfigurer.DefaultCodecs）：跨包，QN 错误 miss；
 *   - 通配符 + 全局精确 QN：miss。
 *   四级检索全 miss -> ResolveType 兜底产出外部符号 -> IMPLEMENT 被 LevelBalanced
 *   过滤而系统性丢失。
 * 修复后：拆外层 CodecConfigurer（import 命中 ...codec.CodecConfigurer）-> 解析 QN
 * 下拼 DefaultCodecs -> 命中内部嵌套接口，产出内部 IMPLEMENT。
 */
class BaseDefaultCodecs implements CodecConfigurer.DefaultCodecs, CodecConfigurer.DefaultCodecConfig {

    @Override
    public int getMaxInMemorySize() {
        return 256 * 1024;
    }

    @Override
    public boolean isEnableLoggingRequestDetails() {
        return false;
    }
}
