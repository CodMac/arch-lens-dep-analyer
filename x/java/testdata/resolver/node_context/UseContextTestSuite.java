package com.example.rel.use;

public class UseContextTestSuite {

    // 1. 局部变量读取在二元表达式中 - unique variable: binaryLocal
    public void useCase001_BinaryExpression() {
        int binaryLocal = 5;
        int result = binaryLocal + 2; // line 8
    }

    // 2. 显式成员变量读取在字段访问中 - unique fields: fieldAccessField
    private int fieldAccessField = 10;
    public void useCase002_FieldAccess() {
        int x = this.fieldAccessField; // line 14
    }

    // 3. 隐式成员变量读取在二元表达式中 - unique fields: implicitField, param: implicitParam
    private int implicitField = 20;
    public void useCase003_ImplicitField(int implicitParam) {
        this.implicitField = implicitParam; // line 20
    }

    // 4. 静态常量读取在方法调用中 - unique constant: staticConstant
    public static final String staticConstant = "FIXED";
    public void useCase004_StaticConstant() {
        System.out.println(staticConstant); // line 26
    }

    // 5. 数组元素读取在数组访问中 - unique variable: arrayVar
    public void useCase005_ArrayAccess() {
        String[] arrayVar = new String[]{"test"};
        String s = arrayVar[0]; // line 32
    }

    // 6. 局部变量作为方法参数 - unique variable: paramVar, method: paramMethod
    public void useCase006_MethodParameter() {
        int paramVar = 10;
        paramMethod(paramVar); // line 38
    }
    public void paramMethod(int value) {}

    // 7. 局部变量在三元表达式中 - unique variable: ternaryVar
    public void useCase007_TernaryExpression() {
        int ternaryVar = 5;
        int val = (ternaryVar > 0) ? ternaryVar : 0; // line 45
    }

    // 8. 集合读取在增强for循环中 - unique variable: collectionVar
    import java.util.List;
    public void useCase008_EnhancedForLoop() {
        List<String> collectionVar = List.of("A", "B");
        for (String item : collectionVar) { // line 52
            System.out.println(item);
        }
    }

    // 9. 字段变量在Lambda捕获中 - unique fields: lambdaField
    private int lambdaField = 30;
    public void useCase009_LambdaCapture() {
        Runnable r = () -> {
            System.out.println(lambdaField); // line 61
        };
    }

    // 10. 对象在类型转换中 - unique variable: castVar
    public void useCase010_TypeCast() {
        Object castVar = "string";
        String str = (String) castVar; // line 68
    }

    // 11. 嵌套链式调用中的参数读取 - unique variables: chainVar, chainMethod
    public void useCase011_ChainedMethod() {
        int chainVar = 100;
        chainMethod(chainVar).chainMethod2(); // line 74
    }
    public ChainClass chainMethod(int value) { return new ChainClass(); }
    public class ChainClass {
        public ChainClass chainMethod2() { return this; }
    }
}