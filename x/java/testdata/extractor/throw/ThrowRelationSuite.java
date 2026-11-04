package com.example.rel;

import java.io.*;
import java.sql.SQLException;

public class ThrowRelationSuite {

    // 1. 方法签名中的声明
    // Source: com.example.rel.ThrowRelationSuite.readFile
    // Target: java.io.IOException
    // Mores: { "java.rel.throw.is_signature": true, "java.rel.throw.index": 0 }
    //
    // Source: com.example.rel.ThrowRelationSuite.readFile
    // Target: java.sql.SQLException
    // Mores: { "java.rel.throw.is_signature": true, "java.rel.throw.index": 1 }
    public void readFile() throws IOException, SQLException {

        // 2. 方法体内主动抛出
        // Source: com.example.rel.ThrowRelationSuite.readFile
        // Target: java.lang.RuntimeException
        // Mores: { "java.rel.raw_text": "throw new RuntimeException(\"Error\")" }
        if (true) throw new RuntimeException("Error");
    }

    // 3. 构造函数声明抛出
    // Source: com.example.rel.ThrowRelationSuite.<init>
    // Target: java.lang.Exception
    // Mores: { "java.rel.throw.is_signature": true }
    public ThrowRelationSuite() throws Exception {}

    // 4. 重新抛出捕获的异常
    // Source: com.example.rel.ThrowRelationSuite.rethrow
    // Target: java.lang.Exception
    public void rethrow() throws Exception {
        try {
            // ...
        } catch (Exception e) {
            throw e;
        }
    }

    // 5. 重新抛出多异常联合类型捕获的异常（catch_type 含 |，取第一个类型 IOException）
    // Source: com.example.rel.ThrowRelationSuite.rethrowMulti
    // Target: java.io.IOException
    // Mores: { "java.rel.throw.is_rethrow": true }
    public void rethrowMulti() throws Exception {
        try {
            // ...
        } catch (IOException | SQLException e) {
            throw e;
        }
    }

    // 6. 方法调用链抛异常（C-9）：throw factory() 返回内部自定义异常类型
    // Source: com.example.rel.ThrowRelationSuite.throwViaFactory
    // Target: com.example.rel.ThrowRelationSuite.MyException
    // Mores: { "java.rel.raw_text": "throw myException()" }
    public void throwViaFactory() {
        throw myException();
    }

    // 7. 方法调用链抛异常（C-9 带参形态）：throw factory(msg, cause) 返回内部异常类型
    // Source: com.example.rel.ThrowRelationSuite.throwViaFactoryWithArgs
    // Target: com.example.rel.ThrowRelationSuite.MyException
    public void throwViaFactoryWithArgs(String msg) {
        throw myException(msg, new RuntimeException());
    }

    private MyException myException() {
        return new MyException();
    }

    private MyException myException(String msg, RuntimeException cause) {
        return new MyException();
    }

    static class MyException extends RuntimeException {
    }
}


