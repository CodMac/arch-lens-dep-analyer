package com.example.rel.call;

public class CallContextTestSuite {

    // 1. 简单方法调用 - unique variable: callVar, method: simpleMethod
    public void callCase001_SimpleMethod() {
        CallHelper callVar = new CallHelper();
        callVar.simpleMethod(); // line 8
    }

    // 2. 静态方法调用 - unique class: StaticCaller, method: staticCallMethod
    public void callCase002_StaticMethod() {
        StaticCaller.staticCallMethod(); // line 13
    }
    public static class StaticCaller {
        public static void staticCallMethod() {}
    }

    // 3. 链式方法调用 - unique variable: chainVar, methods: firstMethod, secondMethod, thirdMethod
    public void callCase003_ChainedMethod() {
        ChainHelper chainVar = new ChainHelper();
        chainVar.firstMethod().secondMethod().thirdMethod(); // line 22
    }
    public static class ChainHelper {
        public ChainHelper firstMethod() { return this; }
        public ChainHelper secondMethod() { return this; }
        public ChainHelper thirdMethod() { return this; }
    }

    // 4. 方法调用作为参数 - unique variables: param1, param2, method: paramMethod
    public void callCase004_MethodAsParameter() {
        int param1 = 10;
        int param2 = 20;
        processResult(calculateSum(param1, param2)); // line 34
    }
    public int calculateSum(int a, int b) { return a + b; }
    public void processResult(int result) {}

    // 5. 构造函数调用 - unique class: NewInstanceClass
    public void callCase005_Constructor() {
        NewInstanceClass instance = new NewInstanceClass(); // line 41
    }
    public static class NewInstanceClass {}

    // 6. 带参数的方法调用 - unique variables: arg1, arg2, arg3, method: threeArgMethod
    public void callCase006_ThreeArguments() {
        int arg1 = 1;
        int arg2 = 2;
        int arg3 = 3;
        threeArgMethod(arg1, arg2, arg3); // line 50
    }
    public void threeArgMethod(int a, int b, int c) {}

    // 7. Lambda内方法调用 - unique variable: lambdaVar, method: lambdaCalledMethod
    public void callCase007_LambdaMethod() {
        int lambdaVar = 100;
        Runnable r = () -> {
            lambdaCalledMethod(lambdaVar); // line 58
        };
    }
    public void lambdaCalledMethod(int value) {}

    // 8. 泛型方法调用 - unique variable: genericVar, method: genericMethod
    public void callCase008_GenericMethod() {
        String genericVar = "test";
        genericMethod(genericVar); // line 66
    }
    public <T> void genericMethod(T value) {}

    // 9. 字段访问后的方法调用 - unique fields: fieldObj, method: fieldMethod
    private FieldClass fieldObj = new FieldClass();
    public void callCase009_FieldThenMethod() {
        fieldObj.fieldMethod(); // line 73
    }
    public static class FieldClass {
        public void fieldMethod() {}
    }

    // 10. 嵌套对象方法调用 - unique objects: outerObj, middleObj, innerObj, method: nestedMethod
    public void callCase010_NestedObjectMethod() {
        OuterObj outerObj = new OuterObj();
        outerObj.middleObj.innerObj.nestedMethod(); // line 82
    }
    public static class OuterObj {
        public MiddleClass middleObj = new MiddleClass();
    }
    public static class MiddleClass {
        public InnerClass innerObj = new InnerClass();
    }
    public static class InnerClass {
        public void nestedMethod() {}
    }

    // 11. 返回对象的方法调用 - unique variable: returnVar, method: getHelper, method: helperMethod
    public void callCase011_ReturnedObjectMethod() {
        returnVar getHelper = getHelper();
        getHelper.helperMethod(); // line 97
    }
    public returnVar getHelper() {
        return new returnVar();
    }
    public static class returnVar {
        public void helperMethod() {}
    }

    // 12. 数组元素作为方法参数 - unique variable: arrayParam, method: arrayMethod
    public void callCase012_ArrayElementParameter() {
        int[] arrayParam = new int[]{1, 2, 3};
        arrayMethod(arrayParam[0]); // line 109
    }
    public void arrayMethod(int value) {}
}