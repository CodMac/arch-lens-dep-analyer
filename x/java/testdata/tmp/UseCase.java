package com.example.resolver.use;

import java.util.List;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;

/**
 * Use关系复杂场景测试用例
 * 涵盖各种变量、字段访问和使用模式
 */
public class UseCase {

    // ==================== 基础定义 ====================

    public String stringValue;
    public int intValue;
    public List<String> stringList;
    public Map<String, String> stringMap;
    public InnerClass innerClass;

    public static class InnerClass {
        public String innerField;
        public NestedClass nestedClass;

        public String getInnerField() {
            return innerField;
        }

        public NestedClass getNestedClass() {
            return nestedClass;
        }
    }

    public static class NestedClass {
        public String nestedField;

        public String getNestedField() {
            return nestedField;
        }
    }

    // ==================== 简单变量使用场景 ====================

    /**
     * 场景1: 简单变量使用 - 算术表达式
     */
    public void simpleVariableUseInArithmetic() {
        int a = 10;
        int b = 20;

        // 简单的变量使用
        int sum = a + b;           // USE: a, b
        int diff = a - b;          // USE: a, b
        int product = a * b;       // USE: a, b
        int quotient = a / b;      // USE: a, b
        int remainder = a % b;     // USE: a, b
    }

    /**
     * 场景2: 简单变量使用 - 比较表达式
     */
    public void simpleVariableUseInComparison() {
        int x = 5;
        int y = 10;

        // 比较表达式中的变量使用
        boolean isEqual = x == y;    // USE: x, y
        boolean isNotEqual = x != y; // USE: x, y
        boolean isGreater = x > y;   // USE: x, y
        boolean isLess = x < y;      // USE: x, y
    }

    /**
     * 场景3: 简单变量使用 - 方法参数
     */
    public void simpleVariableUseInParameter() {
        String message = "Hello";

        // 方法参数中的变量使用
        printMessage(message);        // USE: message
        processString(message);       // USE: message
    }

    // ==================== 字段访问使用场景 ====================

    /**
     * 场景4: 实例字段访问
     */
    public void instanceFieldAccess() {
        this.stringValue = "test";
        this.intValue = 42;

        // 实例字段使用
        String upper = this.stringValue.toUpperCase();  // USE: this.stringValue
        int doubled = this.intValue * 2;              // USE: this.intValue
    }

    /**
     * 场景5: 静态字段访问
     */
    public void staticFieldAccess() {
        // 静态字段使用
        double pi = Math.PI;                    // USE: Math.PI
        String separator = File.separator;       // USE: File.separator
    }

    /**
     * 场景6: 深层字段访问
     */
    public void deepFieldAccess() {
        this.innerClass = new InnerClass();
        this.innerClass.innerField = "deep";
        this.innerClass.nestedClass = new NestedClass();
        this.innerClass.nestedClass.nestedField = "nested";

        // 深层字段使用
        String field1 = this.innerClass.innerField;                     // USE: this.innerClass
        String field2 = this.innerClass.nestedClass.nestedField;         // USE: this.innerClass
        String result = this.innerClass.getNestedClass().getNestedField(); // USE: this.innerClass
    }

    // ==================== 链式访问使用场景 ====================

    /**
     * 场景7: 字段访问链 - obj.field1.field2.field3
     */
    public void fieldAccessChain() {
        this.innerClass = new InnerClass();
        this.innerClass.nestedClass = new NestedClass();
        this.innerClass.nestedClass.nestedField = "chain";

        // 字段访问链
        String chained = this.innerClass.nestedClass.nestedField;  // USE: this.innerClass
    }

    /**
     * 场景8: 字段访问和方法调用混合链 - obj.field1.method().field2
     */
    public void mixedAccessChain() {
        this.innerClass = new InnerClass();
        this.innerClass.innerField = "mixed";
        this.innerClass.nestedClass = new NestedClass();
        this.innerClass.nestedClass.nestedField = "access";

        // 混合访问链
        String field = this.innerClass.getInnerField();                          // USE: this.innerClass
        String nested = this.innerClass.getNestedClass().nestedField;             // USE: this.innerClass
        String method = this.innerClass.getNestedClass().getNestedField();        // USE: this.innerClass
    }

    // ==================== 集合和数组使用场景 ====================

    /**
     * 场景9: 数组元素使用
     */
    public void arrayElementUse() {
        int[] numbers = {1, 2, 3, 4, 5};

        // 数组元素使用
        int first = numbers[0];       // USE: numbers
        int second = numbers[1];      // USE: numbers
        int sum = numbers[0] + numbers[1]; // USE: numbers
    }

    /**
     * 场景10: List集合使用
     */
    public void listCollectionUse() {
        List<String> items = new ArrayList<>();
        items.add("item1");
        items.add("item2");

        // List集合使用
        String firstItem = items.get(0);               // USE: items
        String secondItem = items.get(1);              // USE: items
        int size = items.size();                       // USE: items
    }

    /**
     * 场景11: Map集合使用
     */
    public void mapCollectionUse() {
        Map<String, String> data = new HashMap<>();
        data.put("key1", "value1");
        data.put("key2", "value2");

        // Map集合使用
        String val1 = data.get("key1");    // USE: data
        String val2 = data.get("key2");    // USE: data
        boolean hasKey = data.containsKey("key1"); // USE: data
    }

    // ==================== 复杂表达式使用场景 ====================

    /**
     * 场景12: 复杂算术表达式
     */
    public void complexArithmeticExpression() {
        int a = 10;
        int b = 20;
        int c = 30;

        // 复杂算术表达式
        int complex = (a + b) * c + (a - b) / c;  // USE: a, b, c
        int nested = ((a + b) * (c - a)) + b;      // USE: a, b, c
    }

    /**
     * 场景13: 三元表达式
     */
    public void ternaryExpressionUse() {
        int a = 10;
        int b = 20;
        String str = "test";

        // 三元表达式中的变量使用
        int max = a > b ? a : b;              // USE: a, b
        String result = str.length() > 3 ? str.substring(0, 3) : str; // USE: str
    }

    /**
     * 场景14: 逻辑表达式
     */
    public void logicalExpressionUse() {
        boolean flag1 = true;
        boolean flag2 = false;

        // 逻辑表达式中的变量使用
        boolean andResult = flag1 && flag2;   // USE: flag1, flag2
        boolean orResult = flag1 || flag2;    // USE: flag1, flag2
        boolean notResult = !flag1;           // USE: flag1
    }

    // ==================== 方法调用中的使用场景 ====================

    /**
     * 场景15: 方法调用中的复杂参数
     */
    public void complexMethodParameters() {
        String name = "user";
        int age = 25;
        String city = "Beijing";

        // 方法调用中的复杂参数
        createUser(name, age, city);                          // USE: name, age, city
        createUser(name + " junior", age + 1, city.toUpperCase()); // USE: name, age, city
    }

    /**
     * 场景16: 链式方法调用中的变量使用
     */
    public void chainedMethodCallsUse() {
        List<String> list = new ArrayList<>();
        list.add("hello");
        list.add("world");

        // 链式方法调用中的变量使用
        String result = list.get(0).toUpperCase().concat(" ") + list.get(1); // USE: list
    }

    // ==================== 字符串操作使用场景 ====================

    /**
     * 场景17: 字符串连接和操作
     */
    public void stringOperationUse() {
        String firstName = "John";
        String lastName = "Doe";
        int age = 30;

        // 字符串操作中的变量使用
        String fullName = firstName + " " + lastName;       // USE: firstName, lastName
        String info = firstName + " is " + age + " years old"; // USE: firstName, age
    }

    /**
     * 场景18: 字符串方法调用
     */
    public void stringMethodUse() {
        String text = "Hello World";

        // 字符串方法调用中的变量使用
        String upper = text.toUpperCase();          // USE: text
        String lower = text.toLowerCase();          // USE: text
        String substring = text.substring(0, 5);    // USE: text
        boolean contains = text.contains("Hello");  // USE: text
    }

    // ==================== 条件和循环使用场景 ====================

    /**
     * 场景19: if条件中的变量使用
     */
    public void conditionalVariableUse() {
        int score = 85;

        // 条件语句中的变量使用
        if (score >= 90) {               // USE: score
            System.out.println("Excellent");
        } else if (score >= 70) {        // USE: score
            System.out.println("Good");
        } else {                         // USE: score
            System.out.println("Needs improvement");
        }
    }

    /**
     * 场景20: 循环中的变量使用
     */
    public void loopVariableUse() {
        List<String> items = new ArrayList<>();
        for (int i = 0; i < 10; i++) {  // USE: i
            items.add("item" + i);       // USE: i
        }

        for (String item : items) {      // USE: items
            System.out.println(item);    // USE: item
        }
    }

    /**
     * 场景21: switch语句中的变量使用
     */
    public void switchVariableUse() {
        int day = 3;

        // switch语句中的变量使用
        switch (day) {                   // USE: day
            case 1:
                System.out.println("Monday");
                break;
            case 2:
                System.out.println("Tuesday");
                break;
            case 3:
                System.out.println("Wednesday");
                break;
            default:
                System.out.println("Other day");
        }
    }

    // ==================== Lambda和流式处理使用场景 ====================

    /**
     * 场景22: Lambda表达式中的变量使用
     */
    public void lambdaVariableUse() {
        List<String> names = new ArrayList<>();
        names.add("Alice");
        names.add("Bob");
        names.add("Charlie");

        // Lambda表达式中的变量使用
        names.forEach(name -> System.out.println(name)); // USE: names, name

        names.stream()
             .filter(name -> name.length() > 3)          // USE: names, name
             .forEach(name -> System.out.println(name)); // USE: name
    }

    /**
     * 场景23: 闭包中的外部变量使用
     */
    public void closureVariableUse() {
        String prefix = "User: ";
        List<String> users = new ArrayList<>();
        users.add("John");
        users.add("Jane");

        // 闭包中的外部变量使用
        users.forEach(user -> System.out.println(prefix + user)); // USE: prefix, users, user
    }

    // ==================== 异常处理使用场景 ====================

    /**
     * 场景24: 异常处理中的变量使用
     */
    public void exceptionHandlingUse() {
        String message = "An error occurred";

        try {
            // 某些操作
            processSomething();
        } catch (Exception e) {      // USE: e
            System.out.println(message + ": " + e.getMessage()); // USE: message, e
        }
    }

    /**
     * 场景25: 自定义异常抛出中的变量使用
     */
    public void customExceptionThrowUse() {
        String errorMessage = "Custom error";
        int errorCode = 404;

        // 自定义异常中的变量使用
        if (errorCode == 404) {       // USE: errorCode
            throw new RuntimeException(errorMessage); // USE: errorMessage
        }
    }

    // ==================== 类型转换使用场景 ====================

    /**
     * 场景26: 类型转换中的变量使用
     */
    public void typeConversionUse() {
        Object obj = "Hello";
        Number num = 100;

        // 类型转换中的变量使用
        String str = (String) obj;   // USE: obj
        int value = num.intValue();  // USE: num
    }

    /**
     * 场景27: instanceof中的变量使用
     */
    public void instanceofUse() {
        Object obj = "Hello World";

        // instanceof中的变量使用
        if (obj instanceof String) {  // USE: obj
            String str = (String) obj; // USE: obj
            System.out.println(str.toUpperCase()); // USE: str
        }
    }

    // ==================== 泛型使用场景 ====================

    /**
     * 场景28: 泛型集合使用
     */
    public void genericCollectionUse() {
        List<String> stringList = new ArrayList<>();
        List<Integer> intList = new ArrayList<>();

        stringList.add("test");
        intList.add(42);

        // 泛型集合使用
        String str = stringList.get(0); // USE: stringList
        Integer num = intList.get(0);   // USE: intList
    }

    /**
     * 场景29: 泛型方法中的变量使用
     */
    public <T> void genericMethodUse(T item) {
        // 泛型方法中的参数使用
        processItem(item); // USE: item
    }

    // ==================== 辅助方法 ====================

    public void printMessage(String message) {
        System.out.println(message);
    }

    public void processString(String str) {
        System.out.println("Processing: " + str);
    }

    public void createUser(String name, int age, String city) {
        System.out.println("User: " + name + ", Age: " + age + ", City: " + city);
    }

    public void processSomething() throws Exception {
        // 模拟处理
    }

    public <T> void processItem(T item) {
        System.out.println("Item: " + item);
    }
}