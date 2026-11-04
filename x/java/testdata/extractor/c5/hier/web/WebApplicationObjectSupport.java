package com.example.rel.c5.hier.web;

import com.example.rel.c5.hier.base.ApplicationObjectSupport;

/**
 * WebApplicationObjectSupport 模拟 spring 的 org.springframework.web.context
 * .support.WebApplicationObjectSupport（继承链第 2 级，本文件 import 了基类）。
 */
public class WebApplicationObjectSupport extends ApplicationObjectSupport {

    private String servletContext;

    public void setServletContext(String context) {
        this.servletContext = context;
    }
}
