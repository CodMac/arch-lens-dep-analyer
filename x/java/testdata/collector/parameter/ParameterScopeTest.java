package com.example.base.test;

import java.util.List;
import java.util.Map;

public class ParameterScopeTest {
    // 构造函数
    public ParameterScopeTest(String initialConfig) {
    }

    // 普通方法与变长参数
    public void execute(int times, String... labels) {
    }

    // 泛型数组 / 泛型变长参数：剥泛型实参后必须保留 [] 与 ...（否则重载被压平）
    public void genericArray(Class<?>[] types) {
    }

    public void genericArray(Class<?> type) {
    }

    public void genericVarargs(Class<?>... types) {
    }

    public void genericNested(List<Integer>[] graph, Map<String, List<Integer>>[] index) {
    }

    public void genericVarargsNested(List<Integer>... lists) {
    }

    // 内部类中的方法参数
    class InnerWorker {
        void doWork(long duration) {
        }
    }
}
