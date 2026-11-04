package com.example.rel.c5.hier.base;

/**
 * ApplicationObjectSupport 模拟 spring 的 org.springframework.context.support
 * .ApplicationObjectSupport（3 级继承链最底层基类）。
 * 只在 web.WebApplicationObjectSupport.java 中 import，调用方不 import。
 */
public class ApplicationObjectSupport {

    private Object applicationContext;

    protected void initApplicationContext() {
        this.applicationContext = new Object();
    }

    public Object getApplicationContext() {
        return this.applicationContext;
    }
}
