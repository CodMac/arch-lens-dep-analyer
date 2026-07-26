package com.test.case1;

public class ExceptionFactory {
    public static CustomException createException() {
        return new CustomException("factory created");
    }
}