package com.example.resolver.assign;

import java.util.List;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Map;

/**
 * Assign关系复杂场景测试用例
 * 涵盖各种赋值、字段修改和更新模式
 */
public class AssignCase {

    // ==================== 基础定义 ====================

    public String stringValue;
    public int intValue;
    public double doubleValue;
    public boolean booleanValue;
    public List<String> stringList;
    public Map<String, String> stringMap;
    public InnerClass innerClass;
    public static int staticCount;

    public static class InnerClass {
        public String innerString;
        public int innerInt;
        public NestedClass nestedClass;

        public void setInnerString(String str) {
            this.innerString = str;
        }

        public NestedClass getNestedClass() {
            return nestedClass;
        }
    }

    public static class NestedClass {
        public String nestedField;
        public int nestedValue;

        public void setNestedField(String field) {
            this.nestedField = field;
        }
    }

    // ==================== 简单赋值场景 ====================

    /**
     * 场景1: 简单变量赋值
     */
    public void simpleVariableAssign() {
        int a;
        String s;
        boolean flag;

        // 简单赋值
        a = 10;               // ASSIGN: a
        s = "hello";          // ASSIGN: s
        flag = true;          // ASSIGN: flag
    }

    /**
     * 场景2: 简单字段赋值
     */
    public void simpleFieldAssign() {
        // 字段赋值
        this.stringValue = "field";   // ASSIGN: this.stringValue
        this.intValue = 42;           // ASSIGN: this.intValue
        this.booleanValue = false;    // ASSIGN: this.booleanValue
    }

    /**
     * 场景3: 静态字段赋值
     */
    public void staticFieldAssign() {
        // 静态字段赋值
        staticCount = 100;            // ASSIGN: staticCount
    }

    /**
     * 场景4: 方法调用结果的赋值
     */
    public void methodCallResultAssign() {
        // 方法调用结果赋值
        String result = getValue();           // ASSIGN: result
        int count = getCount();               // ASSIGN: count
        String upper = result.toUpperCase();  // ASSIGN: upper
    }

    // ==================== 复合赋值场景 ====================

    /**
     * 场景5: 复合算术赋值运算符
     */
    public void compoundArithmeticAssign() {
        int a = 10;
        int b = 5;

        // 复合算术赋值
        a += b;      // ASSIGN: a (a = a + b)
        a -= b;      // ASSIGN: a (a = a - b)
        a *= b;      // ASSIGN: a (a = a * b)
        a /= b;      // ASSIGN: a (a = a / b)
        a %= b;      // ASSIGN: a (a = a % b)
    }

    /**
     * 场景6: 位运算复合赋值运算符
     */
    public void bitwiseCompoundAssign() {
        int a = 10;
        int b = 2;

        // 位运算复合赋值
        a &= b;      // ASSIGN: a (a = a & b)
        a |= b;      // ASSIGN: a (a = a | b)
        a ^= b;      // ASSIGN: a (a = a ^ b)
        a <<= b;     // ASSIGN: a (a = a << b)
        a >>= b;     // ASSIGN: a (a = a >> b)
        a >>>= b;    // ASSIGN: a (a = a >>> b)
    }

    /**
     * 场景7: 字符串拼接赋值
     */
    public void stringConcatenationAssign() {
        String message = "Hello";

        // 字符串拼接赋值
        message += " World";    // ASSIGN: message (message = message + " World")
        message += "!";         // ASSIGN: message
    }

    // ==================== 自增自减场景 ====================

    /**
     * 场景8: 自增自减 - 递增
     */
    public void incrementAssign() {
        int counter = 0;

        // 后置自增
        counter++;     // ASSIGN: counter (先取值再自增)

        // 前置自增
        ++counter;     // ASSIGN: counter (先自增再取值)
    }

    /**
     * 场景9: 自增自减 - 递减
     */
    public void decrementAssign() {
        int counter = 10;

        // 后置自减
        counter--;     // ASSIGN: counter (先取值再自减)

        // 前置自减
        --counter;     // ASSIGN: counter (先自减再取值)
    }

    /**
     * 场景10: 字段的自增自减
     */
    public void fieldIncrementDecrement() {
        this.intValue = 100;

        // 字段自增自减
        this.intValue++;     // ASSIGN: this.intValue
        ++this.intValue;     // ASSIGN: this.intValue
        this.intValue--;     // ASSIGN: this.intValue
        --this.intValue;     // ASSIGN: this.intValue
    }

    // ==================== 声明时初始化场景 ====================

    /**
     * 场景11: 变量声明时初始化
     */
    public void declarationWithInitialization() {
        // 声明时初始化
        int age = 25;            // ASSIGN: age
        String name = "John";    // ASSIGN: name
        boolean active = true;   // ASSIGN: active
    }

    /**
     * 场景12: 字段声明时初始化
     */
    public void fieldDeclarationInitialization() {
        // 局部字段声明时初始化
        int localValue = 100;    // ASSIGN: localValue
        String localStr = "local"; // ASSIGN: localStr
    }

    /**
     * 场景13: 集合声明时初始化
     */
    public void collectionDeclarationInitialization() {
        // 集合声明时初始化
        List<String> list = new ArrayList<>();     // ASSIGN: list
        Map<String, String> map = new HashMap<>(); // ASSIGN: map
    }

    // ==================== 数组和集合元素赋值场景 ====================

    /**
     * 场景14: 数组元素赋值
     */
    public void arrayElementAssign() {
        int[] numbers = new int[5];
        String[] strings = new String[3];

        // 数组元素赋值
        numbers[0] = 10;         // ASSIGN: numbers[0]
        numbers[1] = 20;         // ASSIGN: numbers[1]
        strings[0] = "first";    // ASSIGN: strings[0]
        strings[1] = "second";   // ASSIGN: strings[1]
    }

    /**
     * 场景15: 集合元素赋值
     */
    public void collectionElementAssign() {
        List<String> list = new ArrayList<>();
        Map<String, String> map = new HashMap<>();

        // List元素赋值
        list.add("item1");       // CALL: list (通过方法调用)
        list.add("item2");       // CALL: list

        // Map元素赋值
        map.put("key1", "value1"); // CALL: map (通过方法调用)
        map.put("key2", "value2"); // CALL: map
    }

    /**
     * 场景16: 数组元素复合赋值
     */
    public void arrayElementCompoundAssign() {
        int[] numbers = {1, 2, 3};

        // 数组元素复合赋值
        numbers[0] += 10;        // ASSIGN: numbers[0]
        numbers[1] *= 2;         // ASSIGN: numbers[1]
        numbers[2] %= 3;         // ASSIGN: numbers[2]
    }

    // ==================== 链式字段赋值场景 ====================

    /**
     * 场景17: 深层字段赋值
     */
    public void deepFieldAssign() {
        this.innerClass = new InnerClass();
        this.innerClass.nestedClass = new NestedClass();

        // 深层字段赋值
        this.innerClass.innerString = "outer";                        // ASSIGN: this.innerClass.innerString
        this.innerClass.nestedClass.nestedField = "nested";           // ASSIGN: this.innerClass.nestedClass.nestedField
        this.innerClass.nestedClass.nestedValue = 100;                // ASSIGN: this.innerClass.nestedClass.nestedValue
    }

    /**
     * 场景18: 链式字段赋值
     */
    public void chainedFieldAssign() {
        this.innerClass = new InnerClass();
        this.innerClass.nestedClass = new NestedClass();

        // 链式字段赋值
        this.innerClass.setInnerString("update");                      // ASSIGN: this.innerClass
        this.innerClass.getNestedClass().setNestedField("deep");       // ASSIGN: this.innerClass
    }

    /**
     * 场景19: 复杂链式字段赋值
     */
    public void complexChainedFieldAssign() {
        this.innerClass = new InnerClass();

        // 复杂链式字段赋值
        this.innerClass.innerString = "initial";        // ASSIGN: this.innerClass.innerString
        this.innerClass.setInnerString("modified");     // ASSIGN: this.innerClass
    }

    // ==================== 条件和循环中的赋值场景 ====================

    /**
     * 场景20: 条件表达式中的赋值
     */
    public void conditionalExpressionAssign() {
        int a = 10;
        int b = 20;

        // 条件表达式中的赋值
        int max = a > b ? a : b;           // ASSIGN: max
        String result = a > 5 ? "big" : "small"; // ASSIGN: result
    }

    /**
     * 场景21: 条件语句中的赋值
     */
    public void conditionalStatementAssign() {
        int score = 85;
        String grade;

        // 条件语句中的赋值
        if (score >= 90) {
            grade = "A";                   // ASSIGN: grade
        } else if (score >= 70) {
            grade = "B";                   // ASSIGN: grade
        } else {
            grade = "C";                   // ASSIGN: grade
        }
    }

    /**
     * 场景22: 循环中的赋值
     */
    public void loopAssign() {
        int sum = 0;
        int product = 1;

        // 循环中的赋值
        for (int i = 0; i < 10; i++) {    // ASSIGN: i
            sum += i;                     // ASSIGN: sum
            product *= i;                 // ASSIGN: product
        }
    }

    /**
     * 场景23: foreach循环中的赋值
     */
    public void forEachLoopAssign() {
        List<String> items = new ArrayList<>();
        items.add("item1");
        items.add("item2");

        // foreach循环中的变量赋值
        for (String item : items) {        // ASSIGN: item
            System.out.println(item);      // USE: item
        }
    }

    // ==================== 方法参数中的赋值场景 ====================

    /**
     * 场景24: 方法参数链式构造
     */
    public void methodParameterChainedAssign() {
        String base = "user";
        String role = "admin";

        // 方法参数中的链式操作和赋值
        String username = buildUsername(base, role); // ASSIGN: username
        String email = base + "@example.com";        // ASSIGN: email
    }

    /**
     * 场景25: 复杂表达式作为赋值源
     */
    public void complexExpressionAssignSource() {
        int a = 10;
        int b = 20;
        int c = 5;

        // 复杂表达式作为赋值源
        int result1 = (a + b) * c;            // ASSIGN: result1
        int result2 = a > b ? a : b;         // ASSIGN: result2
        String result3 = (a > b ? "big" : "small").toUpperCase(); // ASSIGN: result3
    }

    // ==================== try-catch中的赋值场景 ====================

    /**
     * 场景26: try-catch中的赋值
     */
    public void tryCatchAssign() {
        String message;
        int errorCode = 0;

        try {
            message = "Success";                 // ASSIGN: message
            processOperation();
        } catch (Exception e) {                  // ASSIGN: e
            message = "Error: " + e.getMessage(); // ASSIGN: message
            errorCode = 500;                     // ASSIGN: errorCode
        }
    }

    /**
     * 场景27: finally块中的赋值
     */
    public void finallyBlockAssign() {
        String status = "unknown";

        try {
            status = "processing";    // ASSIGN: status
            doWork();
        } catch (Exception e) {
            status = "failed";       // ASSIGN: status
        } finally {
            status = "completed";     // ASSIGN: status
        }
    }

    // ==================== Lambda和闭包赋值场景 ====================

    /**
     * 场景28: Lambda中的外部变量修改（实际需要final或effectively final）
     */
    public void lambdaVariableModify() {
        List<String> list = new ArrayList<>();
        list.add("a");
        list.add("b");

        final int multiplier = 2;   // ASSIGN: multiplier

        // Lambda处理（注意：Java中lambda中的变量需要是final或effectively final）
        list.stream()
            .map(s -> s + s.length())  // 这里不是赋值，只是使用
            .forEach(s -> System.out.println(s));
    }

    /**
     * 场景29: 闭包中的计数器
     */
    public void closureCounter() {
        // 使用数组来实现闭包中的可变变量
        final int[] counter = {0};        // ASSIGN: counter
        List<String> items = new ArrayList<>(); // ASSIGN: items

        items.add("item1");
        items.add("item2");

        items.forEach(item -> {
            counter[0]++;                 // ASSIGN: counter[0]
            System.out.println(item + ": " + counter[0]);
        });
    }

    // ==================== 嵌套和递归赋值场景 ====================

    /**
     * 场景30: 嵌套赋值表达式（不推荐使用）
     */
    public void nestedAssignment() {
        int a, b, c;

        // 嵌套赋值表达式
        a = b = c = 10;           // ASSIGN: a, b, c (从右到左赋值)
        System.out.println("a=" + a + ", b=" + b + ", c=" + c);
    }

    /**
     * 场景31: 递归方法中的赋值
     */
    public int recursiveAssign(int n) {
        if (n <= 0) {
            return 0;
        }

        int result = n + recursiveAssign(n - 1);  // ASSIGN: result
        return result;
    }

    // ==================== 多重赋值场景 ====================

    /**
     * 场景32: 同一个变量的多重赋值
     */
    public void multipleReassignment() {
        int counter = 0;

        // 多重赋值
        counter = 10;      // ASSIGN: counter
        counter = 20;      // ASSIGN: counter
        counter = 30;      // ASSIGN: counter
        counter += 5;      // ASSIGN: counter
        counter *= 2;      // ASSIGN: counter
    }

    /**
     * 场景33: 多个变量的条件赋值
     */
    public void multipleConditionalAssign() {
        int a = 10;
        int b = 20;
        String result1, result2, result3;

        // 多个变量的条件赋值
        if (a > b) {
            result1 = "a is greater";      // ASSIGN: result1
            result2 = "b is smaller";      // ASSIGN: result2
            result3 = "comparison done";    // ASSIGN: result3
        } else {
            result1 = "b is greater";      // ASSIGN: result1
            result2 = "a is smaller";      // ASSIGN: result2
            result3 = "comparison done";    // ASSIGN: result3
        }
    }

    // ==================== 字符串构建赋值场景 ====================

    /**
     * 场景34: StringBuilder中的附加赋值
     */
    public void stringBuilderAppendAssign() {
        StringBuilder sb = new StringBuilder();

        // StringBuilder中的附加操作（这些不是直接赋值，而是方法调用）
        sb.append("Hello");        // 实际是方法调用，不是直接赋值
        sb.append(" ");
        sb.append("World");

        String result = sb.toString(); // ASSIGN: result
    }

    // ==================== Switch语句中的赋值场景 ====================

    /**
     * 场景35: Switch语句中的赋值
     */
    public void switchStatementAssign() {
        int day = 3;
        String dayName;

        // Switch语句中的赋值
        switch (day) {                  // USE: day
            case 1:
                dayName = "Monday";     // ASSIGN: dayName
                break;
            case 2:
                dayName = "Tuesday";    // ASSIGN: dayName
                break;
            case 3:
                dayName = "Wednesday";  // ASSIGN: dayName
                break;
            default:
                dayName = "Other";      // ASSIGN: dayName
        }
    }

    /**
     * 场景36: Switch表达式中的赋值（Java 12+）
     */
    public void switchExpressionAssign(int score) {
        // Switch表达式赋值
        String grade = switch (score) {  // ASSIGN: grade
            case 90, 91, 92, 93, 94, 95, 96, 97, 98, 99, 100 -> "A";
            case 80, 81, 82, 83, 84, 85, 86, 87, 88, 89 -> "B";
            case 70, 71, 72, 73, 74, 75, 76, 77, 78, 79 -> "C";
            case 60, 61, 62, 63, 64, 65, 66, 67, 68, 69 -> "D";
            default -> "F";
        };
    }

    // ==================== 对象状态修改赋值场景 ====================

    /**
     * 场景37: 对象状态同步修改
     */
    public void objectStateSyncAssign() {
        this.innerClass = new InnerClass();
        this.innerClass.nestedClass = new NestedClass();

        // 同步修改对象状态
        this.innerClass.innerString = "sync1";                 // ASSIGN: this.innerClass.innerString
        this.innerClass.setInnerString("sync2");               // ASSIGN: this.innerClass
        this.innerClass.nestedClass.nestedField = "nestedSync"; // ASSIGN: this.innerClass.nestedClass.nestedField
    }

    // ==================== 辅助方法 ====================

    public String getValue() {
        return "test";
    }

    public int getCount() {
        return 42;
    }

    public String buildUsername(String base, String role) {
        return base + "_" + role;
    }

    public void processOperation() throws Exception {
        // 模拟处理
    }

    public void doWork() throws Exception {
        // 模拟工作
    }
}