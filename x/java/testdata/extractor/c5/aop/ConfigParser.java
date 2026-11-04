package com.example.rel.c5.aop;

import com.example.rel.c5.xml.XmlReaderContext;

/**
 * ConfigParser 模拟 spring 的调用方（如 ConfigBeanDefinitionParser）。
 * 只 import xml.XmlReaderContext，不 import base.ReaderContext。
 *
 * C-5 验证：getReaderContext() 返回 XmlReaderContext（已 import），
 * 其父类 ReaderContext 的实例方法 error()/warning() 应通过继承链解析为
 * 内部方法 CALL（修复前继承链断裂，父类方法 CALL 缺失）。
 */
public class ConfigParser {

    private XmlReaderContext readerContext;

    public ConfigParser(XmlReaderContext readerContext) {
        this.readerContext = readerContext;
    }

    public XmlReaderContext getReaderContext() {
        return this.readerContext;
    }

    public void parse() {
        // C-5 核心：链式方法返回类型的父类方法（跨包继承链第 2 级）。
        // Source: com.example.rel.c5.aop.ConfigParser.parse() (METHOD)
        // Target: com.example.rel.c5.base.ReaderContext.error (METHOD)
        getReaderContext().error("parse failed");

        // C-5 变体：变量形态，父类方法调用（跨包继承链）。
        // Source: com.example.rel.c5.aop.ConfigParser.parse() (METHOD)
        // Target: com.example.rel.c5.base.ReaderContext.warning (METHOD)
        XmlReaderContext ctx = getReaderContext();
        ctx.warning("deprecated");
    }
}
