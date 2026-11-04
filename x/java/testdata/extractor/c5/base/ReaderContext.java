package com.example.rel.c5.base;

/**
 * ReaderContext 模拟 spring 的 org.springframework.beans.factory.parsing.ReaderContext。
 * 继承链最底层基类（第 3 级），只在 xml.XmlReaderContext.java 中 import。
 * 调用方 aop.ConfigParser.java 不 import 本类。
 */
public class ReaderContext {

    private String message;

    public void error(String msg) {
        this.message = msg;
    }

    public void warning(String msg) {
        this.message = msg;
    }
}
