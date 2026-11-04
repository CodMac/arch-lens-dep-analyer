package com.example.rel.c5.hier.web;

import com.example.rel.c5.hier.base.ApplicationObjectSupport;

/**
 * HandlerMapping 模拟 spring 的 org.springframework.web.servlet.HandlerMapping
 * 调用方（3 级继承链场景）。
 *
 * 继承链：HandlerMapping -> WebApplicationObjectSupport -> ApplicationObjectSupport。
 * 调用方只 import web.WebApplicationObjectSupport（第 1 级父类），
 * 基类 ApplicationObjectSupport 只在 WebApplicationObjectSupport.java import。
 *
 * C-5 验证：super.initApplicationContext() 与 this.getApplicationContext() 应
 * 通过跨包 3 级继承链解析为内部方法 CALL（修复前第 2/3 级断链，父类方法缺失）。
 */
public class HandlerMapping extends WebApplicationObjectSupport {

    @Override
    public void setServletContext(String context) {
        super.setServletContext(context);
    }

    public void init() {
        // 3 级继承链：super 方法（定义在 WebApplicationObjectSupport，本文件 import）。
        super.setServletContext("ctx");

        // 3 级继承链：基类方法（ApplicationObjectSupport 未在调用方 import，跨 2 级）。
        // Source: com.example.rel.c5.hier.web.HandlerMapping.init() (METHOD)
        // Target: com.example.rel.c5.hier.base.ApplicationObjectSupport.initApplicationContext (METHOD)
        super.initApplicationContext();

        // 3 级继承链：基类方法（变量形态，跨 2 级）。
        // Source: com.example.rel.c5.hier.web.HandlerMapping.init() (METHOD)
        // Target: com.example.rel.c5.hier.base.ApplicationObjectSupport.getApplicationContext (METHOD)
        Object ac = this.getApplicationContext();
    }
}
