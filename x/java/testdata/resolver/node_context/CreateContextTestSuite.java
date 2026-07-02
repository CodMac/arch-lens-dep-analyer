package com.example.rel.create;

import java.util.List;
import java.util.ArrayList;

public class CreateContextTestSuite {

    // 1. 基本对象创建 - unique class: BasicClass
    public void createCase001_BasicObject() {
        BasicClass basicInstance = new BasicClass(); // line 10
    }
    public static class BasicClass {}

    // 2. 带参数的构造函数 - unique class: ParamClass, variables: paramA, paramB
    public void createCase002_ParameterizedConstructor() {
        int paramA = 1;
        String paramB = "test";
        ParamClass paramInstance = new ParamClass(paramA, paramB); // line 18
    }
    public static class ParamClass {
        public ParamClass(int a, String b) {}
    }

    // 3. 数组创建 - unique variable: arrayInstance
    public void createCase003_ArrayCreation() {
        String[] arrayInstance = new String[10]; // line 26
    }

    // 4. 匿名类创建 - unique variable: anonymousInstance
    public void createCase004_AnonymousClass() {
        Runnable anonymousInstance = new Runnable() { // line 31
            @Override
            public void run() {
                System.out.println("Anonymous class");
            }
        };
    }

    // 5. 泛型对象创建 - unique class: GenericClass, variable: genericInstance
    public void createCase005_GenericType() {
        GenericClass<String> genericInstance = new GenericClass<>(); // line 41
    }
    public static class GenericClass<T> {}

    // 6. 集合对象创建 - unique variable: listInstance
    public void createCase006_CollectionInstance() {
        List<String> listInstance = new ArrayList<>(); // line 47
    }

    // 7. 二维数组创建 - unique variable: twoDArray
    public void createCase007_2DArray() {
        String[][] twoDArray = new String[5][10]; // line 52
    }

    // 8. 数组初始化数组 - unique variable: initArray
    public void createCase008_InitializedArray() {
        String[] initArray = new String[]{1, 2, 3, 4, 5}; // line 57
    }

    // 9. Lambda表达式创建 - unique variable: lambdaInstance
    public void createCase009_LambdaExpression() {
        Runnable lambdaInstance = () -> { // line 62
            System.out.println("Lambda expression");
        };
    }

    // 10. 链式构造对象 - unique variables: outerInstance, innerInstance
    public void createCase010_ChainedCreation() {
        OuterCreator outerInstance = new OuterCreator();
        InnerClass innerInstance = outerInstance.createInner(); // line 70
    }
    public static class OuterCreator {
        public InnerClass createInner() {
            return new InnerClass();
        }
    }
    public static class InnerClass {}

    // 11. 泛型数组创建 - unique variable: genericArray
    public void createCase011_GenericArray() {
        List<String>[] genericArray = new List[10]; // line 81
    }

    // 12. 方法引用创建 - unique variable: methodRefInstance
    public void createCase012_MethodReference() {
        Supplier<String> methodRefInstance = String::new; // line 86
    }
    @FunctionalInterface
    interface Supplier<T> {
        T get();
    }

    // 13. 枚举实例创建 - unique variable: enumInstance
    public void createCase013_EnumInstance() {
        MyEnum enumInstance = MyEnum.ENUM_VALUE; // line 95
    }
    public enum MyEnum {
        ENUM_VALUE
    }

    // 14. 带初始化块的类创建 - unique variable: initBlockInstance
    public void createCase014_InitializationBlock() {
        InitBlockClass initBlockInstance = new InitBlockClass(); // line 103
    }
    public static class InitBlockClass {
        static {
            System.out.println("Static block");
        }
        {
            System.out.println("Instance block");
        }
    }
}