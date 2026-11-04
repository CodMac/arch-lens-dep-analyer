package com.example.rel.c5.xml;

import com.example.rel.c5.base.ReaderContext;

/**
 * XmlReaderContext 模拟 spring 的 org.springframework.beans.factory.xml.XmlReaderContext。
 * 继承跨包基类 ReaderContext（本文件 import 了 ReaderContext，调用方没有）。
 * 链式调用 getReaderContext().error() 的返回类型容器。
 *
 * C-5 缺陷场景：调用方 aop.ConfigParser 解析 getReaderContext().error() 时，
 * 返回类型 XmlReaderContext 的父类 ReaderContext 未在调用方 import ->
 * 继承链断裂 -> 父类方法 error()/warning() 的 CALL 缺失。
 */
public class XmlReaderContext extends ReaderContext {

    public XmlReaderContext(String doc) {
        super();
    }

    public XmlReaderContext getReaderContext() {
        return this;
    }
}
