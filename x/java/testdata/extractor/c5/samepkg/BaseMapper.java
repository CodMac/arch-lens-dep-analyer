package com.example.rel.c5.samepkg;

import com.example.rel.c5.hier.web.WebSupport;

/**
 * 同包不同文件的父类（C-5 回归验证：同包继承不应退化）。
 * 完整模拟 spring AbstractHandlerMapping：自身也有跨包父类 WebSupport + 接口 Matchable。
 * 定义 getPathMatcher() 供子类调用。
 */
public class BaseMapper extends WebSupport implements Matchable {

    private String pathMatcher;

    public String getPathMatcher() {
        return this.pathMatcher;
    }

    public void setPathMatcher(String pm) {
        this.pathMatcher = pm;
    }
}
