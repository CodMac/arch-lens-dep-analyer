package com.example.rel.implement.case1.codec;

/**
 * CodecConfigurer 模拟 spring-web 的 CodecConfigurer（C-10 修复验证）。
 * 外层接口内嵌两个接口（DefaultCodecs / DefaultCodecConfig），供跨包的
 * support.BaseDefaultCodecs 通过点分嵌套名 implements 实现。
 */
public interface CodecConfigurer {

    /** 嵌套接口 1：默认编解码器配置入口。 */
    interface DefaultCodecs {
        int getMaxInMemorySize();
    }

    /** 嵌套接口 2：默认编解码配置项。 */
    interface DefaultCodecConfig {
        boolean isEnableLoggingRequestDetails();
    }

    DefaultCodecs defaultCodecs();
}
