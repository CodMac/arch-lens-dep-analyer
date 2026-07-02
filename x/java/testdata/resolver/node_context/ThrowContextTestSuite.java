package com.example.rel.throw;

import java.io.IOException;
import java.sql.SQLException;

public class ThrowContextTestSuite {

    // 1. 抛出基本异常 - unique variable: basicException
    public void throwCase001_BasicException() throws Exception {
        Exception basicException = new Exception("Basic error");
        throw basicException; // line 11
    }

    // 2. 抛出运行时异常 - unique variable: runtimeException
    public void throwCase002_RuntimeException() {
        RuntimeException runtimeException = new RuntimeException("Runtime error");
        throw runtimeException; // line 17
    }

    // 3. 抛出自定义异常 - unique classes: CustomException, variables: customException
    public void throwCase003_CustomException() throws CustomException {
        CustomException customException = new CustomException("Custom error");
        throw customException; // line 23
    }
    public static class CustomException extends Exception {
        public CustomException(String message) {
            super(message);
        }
    }

    // 4. 条件抛出异常 - unique variables: condition, conditionalException
    public void throwCase004_ConditionalThrow() {
        boolean condition = false;
        if (condition) {
            Exception conditionalException = new Exception("Conditional error");
            throw conditionalException; // line 36
        }
    }

    // 5. Lambda内抛出异常 - unique variable: lambdaException
    public void throwCase005_LambdaThrow() throws Exception {
        Exception lambdaException = new Exception("Lambda error");
        ThrowingInterface thrower = () -> {
            throw lambdaException; // line 44
        };
        thrower.throwMethod();
    }
    @FunctionalInterface
    interface ThrowingInterface {
        void throwMethod() throws Exception;
    }

    // 6. 抛出检查型异常 - unique variable: checkedException
    public void throwCase006_CheckedIOException() throws IOException {
        IOException checkedException = new IOException("IO error");
        throw checkedException; // line 56
    }

    // 7. 抛出链式异常 - unique variables: causeException, wrapperException
    public void throwCase007_ChainedException() throws Exception {
        Exception causeException = new Exception("Cause");
        Exception wrapperException = new Exception("Wrapper", causeException);
        throw wrapperException; // line 63
    }

    // 8. 循环中抛出异常 - unique variables: loopException, loopCounter
    public void throwCase008_LoopThrow() {
        for (int loopCounter = 0; loopCounter < 5; loopCounter++) {
            if (loopCounter == 3) {
                Exception loopException = new Exception("Loop error");
                throw loopException; // line 71
            }
        }
    }

    // 9. 抛出SQL异常 - unique variable: sqlException
    public void throwCase009_SqlException() throws SQLException {
        SQLException sqlException = new SQLException("Database error");
        throw sqlException; // line 79
    }

    // 10. 抛出空指针异常 - unique variable: nullPointerException
    public void throwCase010_NullPointerException() {
        NullPointerException nullPointerException = new NullPointerException("Null pointer error");
        throw nullPointerException; // line 85
    }

    // 11. 抛出非法参数异常 - unique variables: invalidParam, illegalArgException
    public void throwCase011_IllegalArgumentException() {
        String invalidParam = null;
        if (invalidParam == null) {
            IllegalArgumentException illegalArgException = new IllegalArgumentException("Invalid parameter");
            throw illegalArgException; // line 93
        }
    }

    // 12. 抛出断言错误 - unique variable: assertionError
    public void throwCase012_AssertionError() {
        boolean condition = false;
        if (!condition) {
            AssertionError assertionError = new AssertionError("Assertion failed");
            throw assertionError; // line 102
        }
    }

    // 13. 抛出异常状态异常 - unique variable: illegalStateException
    public void throwCase013_IllegalStateException() throws IllegalStateException {
        boolean valid = false;
        if (!valid) {
            IllegalStateException illegalStateException = new IllegalStateException("Invalid state");
            throw illegalStateException; // line 111
        }
    }

    // 14. 抛出类型转换异常 - unique variable: classCastException
    public void throwCase014_ClassCastException() {
        Object obj = "string";
        if (!(obj instanceof Integer)) {
            try {
                Integer integer = (Integer) obj;
            } catch (ClassCastException classCastException) {
                // 不抛出，只是捕获
            }
            ClassCastException classCastException = new ClassCastException("Type cast error");
            throw classCastException; // line 125
        }
    }

    // 15. 匿名类中抛出异常 - unique variable: anonymousException
    public void throwCase015_AnonymousClassThrow() throws Exception {
        Exception anonymousException = new Exception("Anonymous error");
        ExceptionThrower thrower = new ExceptionThrower() {
            @Override
            public void throwIt() throws Exception {
                throw anonymousException; // line 135
            }
        };
        thrower.throwIt();
    }
    interface ExceptionThrower {
        void throwIt() throws Exception;
    }
}