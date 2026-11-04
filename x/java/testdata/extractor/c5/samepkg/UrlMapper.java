package com.example.rel.c5.samepkg;

/**
 * 同包不同文件的子类（模拟 AbstractUrlHandlerMapping extends AbstractHandlerMapping，
 * 二者同包 web.servlet.handler）。
 *
 * C-5 回归验证：同包继承链的方法调用应解析为内部父类方法（修复前依赖调用方 fc
 * 同包 BuildQualifiedName 命中；修复后 ResolveInheritanceParent 不应破坏此路径）。
 */
public class UrlMapper extends BaseMapper {

    public void lookup() {
        // 同包父类方法调用（无接收者，this.getPathMatcher()）。
        // Source: com.example.rel.c5.samepkg.UrlMapper.lookup() (METHOD)
        // Target: com.example.rel.c5.samepkg.BaseMapper.getPathMatcher (METHOD)
        String pm = getPathMatcher();

        // 同包父类方法调用（链式返回类型）。
        // Source: com.example.rel.c5.samepkg.UrlMapper.lookup() (METHOD)
        // Target: com.example.rel.c5.samepkg.BaseMapper.getPathMatcher (METHOD)
        int len = getPathMatcher().length();
    }
}
