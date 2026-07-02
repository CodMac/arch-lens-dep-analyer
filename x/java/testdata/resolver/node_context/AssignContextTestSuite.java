package com.example.rel.assign;

public class AssignContextTestSuite {

    private int instanceField = 0;
    private static int staticField = 0;
    private Data data = new Data();

    public void assignCase001_LocalVariable() {
        int local = 10; // 局部变量普通赋值
        local = 20;     // Line: 11 (左值：identifier)
    }

    public void assignCase002_ExplicitField() {
        this.instanceField = 100; // Line: 15 (左值：field_access 且为链式)
    }

    public void assignCase003_ImplicitField() {
        instanceField = 200; // Line: 19 (左值：identifier，隐式成员变量赋值)
    }

    public void assignCase004_StaticField() {
        AssignContextTestSuite.staticField = 300; // Line: 23 (左值：field_access 且为静态类访问链)
    }

    public void assignCase005_ArrayElement() {
        int[] arr = new int[5];
        arr[0] = 50; // Line: 28 (左值：array_access 且为链式)
    }

    public void assignCase006_ChainedField() {
        this.data.value = 500; // Line: 32 (左值：二级 field_access 深度嵌套链式)
    }

    public void assignCase007_ComplexExpressionNoise() {
        // 带有小括号等语法噪音的复杂左值赋值
        (this.data).value = 999; // Line: 37 (左值：能够安全识别为 field_access)
    }

    private static class Data {
        public int value = 0;
    }
}