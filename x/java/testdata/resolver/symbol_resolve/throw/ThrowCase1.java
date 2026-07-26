package com.test;

import java.io.IOException;
import java.util.Optional;
import com.test.dependency.CustomException;
import com.test.dependency.ExceptionFactory;

public class ThrowCase1 {

    public void execute(Object obj, boolean flag) throws Exception {
        // 维度 1: 基础实例化抛出 (Instantiation Throw)

        // Line 15: 场景 1.1 - 同包/已导入的自定义异常抛出
        if (obj == null) {
            throw new CustomException("object is null");
        }

        // Line 20: 场景 1.2 - 标准库异常抛出
        if (!flag) {
            throw new IllegalArgumentException("invalid flag");
        }

        // Line 25: 场景 1.3 - 未导入/未知异常保底抛出
        if (flag) {
            throw new UnknownException();
        }

        // 维度 2: 变量与参数直接抛出 (Variable / Parameter Throw)


        // Line 35: 场景 2.1 - Catch 块形参直接重新抛出
        try {
            System.out.println("do something");
        } catch (IOException e) {
            throw e;
        }

        // Line 40: 场景 2.2 - 局部变量抛出
        RuntimeException ex = new RuntimeException("runtime error");
        throw ex;
    }

    public void executeAdvanced(Object obj, boolean flag) {
        // 维度 3: 工厂方法与复杂表达式抛出 (Factory Method & Complex Throw)

        // Line 48: 场景 3.1 - 静态工厂方法创建并抛出（推导工厂方法的返回值类型）
        if (obj == null) {
            throw ExceptionFactory.createException();
        }

        // Line 53: 场景 3.2 - 带 Cast 强转的异常抛出
        if (obj instanceof Exception) {
            throw (RuntimeException) obj;
        }


        // 维度 4: 三元运算符抛出 (Ternary Throw)

        // Line 60: 场景 4.1 - 三元表达式分支抛出
        throw flag ? new CustomException() : new IllegalArgumentException();
    }

    public void executeLambda() {
        // 维度 5: Lambda 与 Optional 抛出

        // Line 68: 场景 5.1 - Lambda 作用域内部抛出
        Runnable r = () -> {
            throw new IllegalStateException("lambda error");
        };

        // Line 73: 场景 5.2 - Optional orElseThrow 方法引用抛出
        Optional<String> opt = Optional.empty();
        opt.orElseThrow(CustomException::new);
    }
}