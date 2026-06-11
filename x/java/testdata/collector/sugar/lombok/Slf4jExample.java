package com.example.lombok;

import lombok.extern.slf4j.Slf4j;

@Slf4j
public class Slf4jExample {

    public void logSomething() {
        // log字段应该被识别
        log.debug("This is a debug message");
        log.info("This is an info message");
        log.warn("This is a warning message");
        log.error("This is an error message");

        // 测试变量
        String message = "Hello";
        log.debug("Message: {}", message);
    }
}