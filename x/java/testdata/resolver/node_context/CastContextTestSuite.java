package com.example.rel.cast;

import java.util.List;
import java.util.ArrayList;

public class CastContextTestSuite {

    // 1. 基本类型转换 - unique variables: sourceObj, targetStr
    public void castCase001_BasicCastException() {
        Object sourceObj = "string";
        String targetStr = (String) sourceObj; // line 11
    }

    // 2. 父类转子类 - unique classes: ParentClass, ChildClass, variables: parentVar, childVar
    public void castCase002_Downcast() {
        ParentClass parentVar = new ChildClass();
        ChildClass childVar = (ChildClass) parentVar; // line 17
    }
    public static class ParentClass {}
    public static class ChildClass extends ParentClass {}

    // 3. 接口转实现类 - unique classes: ParentInterface, ImplClass, variables: interfaceVar, implVar
    public void castCase003_InterfaceToImpl() {
        ParentInterface interfaceVar = new ImplClass();
        ImplClass implVar = (ImplClass) interfaceVar; // line 25
    }
    public interface ParentInterface {}
    public static class ImplClass implements ParentInterface {}

    // 4. 数组类型转换 - unique variables: objArray, stringArray
    public void castCase004_ArrayCast() {
        Object[] objArray = new Object[10];
        String[] stringArray = (String[]) objArray; // line 33
    }

    // 5. 链式类型转换 - unique variables: object1, object2, object3
    public void castCase005_ChainedCast() {
        Object object1 = 123;
        Object object2 = (Integer) object1;
        Object object3 = (Integer) object2; // line 40
    }

    // 6. 泛型类型转换 - unique variables: genericObj, specificList
    public void castCase006_GenericCast() {
        Object genericObj = new ArrayList<String>();
        @SuppressWarnings("unchecked")
        List<String> specificList = (List<String>) genericObj; // line 47
    }

    // 7. 原始类型转换 - unique variables: intObj, intVal
    public void castCase007_PrimitiveCast() {
        Integer intObj = Integer.valueOf(123);
        int intVal = (int) intObj; // line 53
    }

    // 8. 方法调用中的类型转换 - unique variables: methodArg, methodResult
    public void castCase008_CastInMethodCall() {
        Object methodArg = "test";
        processString((String) methodArg); // line 59
    }
    public void processString(String str) {}

    // 9. Lambda内的类型转换 - unique variables: lambdaArg, lambdaResult
    public void castCase009_LambdaCast() {
        Object lambdaArg = "lambda";
        Runnable r = () -> {
            String lambdaResult = (String) lambdaArg; // line 67
        };
    }

    // 10. 条件表达式中的类型转换 - unique variables: conditionObj, resultStr
    public void castCase010_CastInTernary() {
        Object conditionObj = "conditional";
        String resultStr = (conditionObj != null) ? (String) conditionObj : "default"; // line 74
    }

    // 11. instanceof检查后的转换 - unique variables: checkObj, castedObj
    public void castCase011_InstanceOfThenCast() {
        Object checkObj = new ChildClass();
        if (checkObj instanceof ChildClass) {
            ChildClass castedObj = (ChildClass) checkObj; // line 81
        }
    }

    // 12. 多重类型转换 - unique variables: multiObj, strObj, intResult
    public void castCase012_MultiLevelCast() {
        Object multiObj = "123";
        String strObj = (String) multiObj;
        int intResult = Integer.parseInt(strObj); // line 89
    }

    // 13. 异常类型转换 - unique variables: exceptionObj, runtimeException
    public void castCase013_ExceptionCast() {
        Exception exceptionObj = new RuntimeException();
        RuntimeException runtimeException = (RuntimeException) exceptionObj; // line 95
    }

    // 14. 数组元素类型转换 - unique variables: elementArray, castedElement
    public void castCase014_ArrayElementCast() {
        Object[] elementArray = new Object[1];
        elementArray[0] = "element";
        String castedElement = (String) elementArray[0]; // line 102
    }

    // 15. 匿名类中的类型转换 - unique variable: anonymousCast
    public void castCase015_AnonymousClassCast() {
        interface CastInterface {
            void castMethod(Object obj);
        }
        CastInterface anonymousCast = (Object obj) -> {
            String casted = (String) obj; // line 111
        };
    }
}