package com.example.resolver.segmenter.use;

import java.util.List;
import java.util.ArrayList;
import java.util.Map;
import java.util.HashMap;
import java.util.Optional;

/**
 * ExpressionSegmenter USE 关系类型全场景测试用例
 * 专门测试变量使用/读取相关的链式表达式解析
 * 覆盖 expression_segmenter.go 中的标识符在各种上下文中的使用场景
 */
public class UseExpressionSegmenterCase {

    // ==================== 辅助类定义 ====================

    public static class Resource {
        public String name;
        public String type;
        public int value;
        public Resource parent;
        public List<Resource> children;
        public Resource[] resources;
        public Map<String, Resource> resourceMap;

        public String getName() {
            return name;
        }

        public Resource getParent() {
            return parent;
        }

        public List<Resource> getChildren() {
            return children;
        }

        public Resource getResource(int index) {
            return resources[index];
        }

        public Resource getResource(String key) {
            return resourceMap.get(key);
        }

        public String getInfo(String prefix) {
            return prefix + name + ":" + value;
        }
    }

    public static class Calculator {
        public int add(int a, int b) {
            return a + b;
        }

        public int multiply(int a, int b) {
            return a * b;
        }

        public int calculate(int a, int b, int c) {
            return a * b + c;
        }
    }

    public static class Formatter {
        public String format(String input, String pattern) {
            return String.format(pattern, input);
        }

        public String truncate(String input, int length) {
            return input.substring(0, Math.min(length, input.length()));
        }
    }

    // ==================== 简单变量使用场景 ====================

    /**
     * 场景1: 简单局部变量使用
     * 对应: localVar
     */
    public void testSimpleVariableUse() {
        String localVar = "value";
        System.out.println(localVar);  // 关键点: localVar

        int count = 100;
        int doubled = count * 2;  // 关键点: count
    }

    /**
     * 场景2: 对象引用使用
     * 对应: objRef
     */
    public void testObjectUse() {
        Resource resource = new Resource();
        System.out.println(resource);  // 关键点: resource

        processResource(resource);  // 关键点: resource
    }

    /**
     * 场景3: 方法返回值使用
     * 对应: obj.method()
     */
    public void testMethodResultUse() {
        Resource resource = new Resource();
        String name = resource.getName();  // 关键点: resource.getName()

        System.out.println(name);  // 关键点: name
    }

    // ==================== 字段使用场景 ====================

    /**
     * 场景4: 简单字段使用
     * 对应: obj.field
     */
    public void testSimpleFieldUse() {
        Resource resource = new Resource();
        String resourceName = resource.name;  // 关键点: resource.name

        int resourceValue = resource.value;  // 关键点: resource.value
    }

    /**
     * 场景5: 嵌套字段使用
     * 对应: obj.field1.field2
     */
    public void testNestedFieldUse() {
        Resource resource = new Resource();
        String parentName = resource.parent.name;  // 关键点: resource.parent.name

        int parentValue = resource.parent.value;  // 关键点: resource.parent.value
    }

    /**
     * 场景6: 深层嵌套字段使用
     * 对应: obj.field1.field2.field3.field4
     */
    public void testDeepNestedFieldUse() {
        Resource resource = new Resource();
        String deepName = resource.parent.parent.name;  // 关键点: resource.parent.parent.name

        int deepValue = resource.parent.children.get(0).value;  // 关键点: resource.parent.children.get(0).value
    }

    // ==================== this 字段使用场景 ====================

    /**
     * 场景7: this 字段使用
     * 对应: this.field
     */
    public void testThisFieldUse() {
        String currentName = this.name;  // 关键点: this.name

        int currentValue = this.value;  // 关键点: this.value
    }

    /**
     * 场景8: this 嵌套字段使用
     * 对应: this.field1.field2
     */
    public void testThisNestedFieldUse() {
        Resource resource = new Resource();
        String rootName = resource.name;  // 关键点: resource.name

        String parentName = resource.parent.name;  // 关键点: resource.parent.name
    }

    // ==================== 方法调用结果使用场景 ====================

    /**
     * 场景9: 方法返回值直接使用
     * 对应: obj.method()
     */
    public void testMethodResultDirectUse() {
        Resource resource = new Resource();
        System.out.println(resource.getName());  // 关键点: resource.getName()

        int length = resource.getName().length();  // 关键点: resource.getName().length()
    }

    /**
     * 场景10: 链式方法调用结果使用
     * 对应: obj.method1().method2()
     */
    public void testChainedMethodResultUse() {
        Resource resource = new Resource();
        String parentName = resource.getParent().getName();  // 关键点: resource.getParent().getName()

        String info = resource.getParent().getInfo("Info: ");  // 关键点: resource.getParent().getInfo("Info: ")
    }

    // ==================== 数组元素使用场景 ====================

    /**
     * 场景11: 数组简单元素使用
     * 对应: obj.array[0]
     */
    public void testArraySimpleElementUse() {
        Resource[] resources = new Resource[10];
        resources[0] = new Resource();

        Resource first = resources[0];  // 关键点: resources[0]
        String firstName = resources[0].name;  // 关键点: resources[0].name
    }

    /**
     * 场景12: 数组嵌套元素使用
     * 对应: obj.array[0].field.array1[1]
     */
    public void testArrayNestedElementUse() {
        Resource[] resources = new Resource[10];
        resources[1] = new Resource();
        resources[1].parent = new Resource();

        String parentName = resources[1].parent.name;  // 关键点: resources[1].parent.name
    }

    /**
     * 场景13: 数组元素在表达式中使用
     * 对应: obj.array[0] + other.array[1]
     */
    public void testArrayElementInExpressionUse() {
        Resource[] resources = new Resource[10];
        int total = resources[0].value + resources[1].value;  // 关键点: resources[0].value + resources[1].value

        String combined = resources[0].name + resources[1].name;  // 关键点: 资源名称拼接
    }

    // ==================== 集合元素使用场景 ====================

    /**
     * 场景14: 集合方法返回元素使用
     * 对应: obj.list.method(0)
     */
    public void testCollectionMethodElementUse() {
        Resource resource = new Resource();
        Resource firstChild = resource.getChildren().get(0);  // 关键点: resource.getChildren().get(0)

        String firstChildName = firstChild.name;  // 关键点: firstChild.name
    }

    /**
     * 场景15: 集合链式调用元素使用
     * 对应: var = obj.method1().method2().get(0)
     */
    public void testChainedCollectionElementUse() {
        Resource resource = new Resource();
        String parentName = resource.getChildren().get(0).getParent().getName();  // 关键点: resource.getChildren().get(0).getParent().getName()

        int childValue = resource.getChildren().get(1).value;  // 关键点: resource.getChildren().get(1).value
    }

    // ==================== Map 操作使用场景 ====================

    /**
     * 场景16: Map get 结果使用
     * 对应: obj.map.get(key)
     */
    public void testMapGetUse() {
        Resource resource = new Resource();
        Resource retrieved = resource.getResource("key1");  // 关键点: resource.getResource("key1")

        String retrievedName = retrieved.name;  // 关键点: retrieved.name
    }

    /**
     * 场景17: Map 链式调用使用
     * 对应: obj.method().get(key).field
     */
    public void testMapChainedUse() {
        Resource resource = new Resource();
        String nestedName = resource.getResource("key").getParent().getName();  // 关键点: resource.getResource("key").getParent().getName()

        int nestedValue = resource.getResource("key").value;  // 关键点: resource.getResource("key").value
    }

    // ==================== 二元表达式中的使用场景 ====================

    /**
     * 场景18: 变量在加法表达式中使用
     * 对应: var + value
     */
    public void testVariableInAdditionUse() {
        int a = 10;
        int b = 20;
        int sum = a + b;  // 关键点: a + b
    }

    /**
     * 场景19: 链式调用在二元表达式中使用
     * 对应: obj.method1().field + obj2.method2()
     */
    public void testChainInBinaryExpressionUse() {
        Resource resource = new Resource();
        int total = resource.value + resource.parent.value;  // 关键点: resource.value + resource.parent.value

        String combined = resource.name + resource.parent.name;  // 关键点: 名称拼接
    }

    /**
     * 场景20: 复杂表达式中的链式使用
     * 对应: obj.method1().field * method2() / method3()
     */
    public void testComplexExpressionChainUse() {
        Resource resource = new Resource();
        int calculated = (resource.value * 2) + (resource.parent.value / 2);  // 关键点: 复杂计算表达式

        Calculator calculator = new Calculator();
        int result = calculator.add(resource.value, resource.parent.value);  // 关键点: 计算器使用
    }

    // ==================== 括号内使用场景 ====================

    /**
     * 场景21: 括号内字段访问使用
     * 对应: (obj.field)
     */
    public void testParenthesizedFieldUse() {
        Resource resource = new Resource();
        String name = (resource.name);  // 关键点: (resource.name)

        int value = (resource.value);  // 关键点: (resource.value)
    }

    /**
     * 场景22: 括号内方法调用使用
     * 对应: (obj.method())
     */
    public void testParenthesizedMethodUse() {
        Resource resource = new Resource();
        String name = (resource.getName());  // 关键点: (resource.getName())

        int length = (resource.getName().length());  // 关键点: (resource.getName().length())
    }

    /**
     * 场景23: 括号内复杂链式使用
     * 对应: ((obj.method1().field))
     */
    public void testNestedParenthesizedUse() {
        Resource resource = new Resource();
        String parentName = ((resource.getParent().getName()));  // 关键点: ((resource.getParent().getName()))

        int parentValue = ((resource.getParent().value));  // 关键点: ((resource.getParent().value))
    }

    // ==================== 方法参数中的使用场景 ====================

    /**
     * 场景24: 简单变量作为方法参数
     * 对应: method(var)
     */
    public void testVariableAsParameterUse() {
        String name = "test";
        System.out.println(name);  // 关键点: name

        int value = 100;
        int doubled = value * 2;  // 关键点: value
    }

    /**
     * 场景25: 字段访问作为方法参数
     * 对应: method(obj.field)
     */
    public void testFieldAsParameterUse() {
        Resource resource = new Resource();
        String formatted = formatResource(resource.name, "Name: %s");  // 关键点: resource.name

        int calculated = calculateValue(resource.value, resource.parent.value);  // 关键点: resource.value + resource.parent.value
    }

    /**
     * 场景26: 链式调用作为方法参数
     * 对应: method(obj.method1().method2())
     */
    public void testChainAsParameterUse() {
        Resource resource = new Resource();
        String info = resource.getParent().getInfo("Parent: ");  // 关键点: resource.getParent().getInfo("Parent: ")

        String formatted = formatResource(resource.getParent().getName(), "Parent: %s");  // 关键点: resource.getParent().getName()
    }

    // ==================== 条件表达式中的使用场景 ====================

    /**
     * 场景27: 变量在条件表达式中使用
     * 对应: condition ? var1 : var2
     */
    public void testVariableInConditionalUse() {
        Resource resource = new Resource();
        String selectedName = resource.parent != null
            ? resource.parent.name
            : resource.name;  // 关键点: 条件表达式中的变量使用
    }

    /**
     * 场景28: 链式调用在条件表达式中使用
     * 对应: condition ? obj1.method1() : obj2.method2()
     */
    public void testChainInConditionalUse() {
        Resource resource = new Resource();
        String selected = resource.getParent() != null
            ? resource.getParent().getName()
            : resource.getName();  // 关键点: 条件表达式中的链式使用

        int selectedValue = resource.getParent() != null
            ? resource.getParent().value
            : resource.value;  // 关键点: 条件表达式中的值选择
    }

    /**
     * 场景29: 复杂条件表达式的使用
     * 对应: (condition && obj.field) ? obj.method1() : obj.method2()
     */
    public void testComplexConditionalUse() {
        Resource resource = new Resource();
        String result = (resource.parent != null && resource.parent.value > 0)
            ? resource.getParent().getName()
            : resource.name;  // 关键点: 复杂条件中的链式使用
    }

    // ==================== Lambda/Stream 中的使用场景 ====================

    /**
     * 场景30: Lambda 表达式中的参数使用
     * 对应: list.forEach(item -> item.method())
     */
    public void testLambdaParameterUse() {
        List<Resource> resources = new ArrayList<>();
        resources.forEach(resource -> System.out.println(resource.name));  // 关键点: resource.name

        resources.forEach(resource -> processResource(resource));  // 关键点: resource
    }

    /**
     * 场景31: Stream map 中的链式使用
     * 对应: list.stream().map(obj -> obj.method())
     */
    public void testStreamMapUse() {
        List<Resource> resources = new ArrayList<>();
        List<String> names = resources.stream()
                                    .map(resource -> resource.getName())
                                    .ToList();  // 关键点: resource.getName()

        List<Integer> values = resources.stream()
                                       .map(resource -> resource.value)
                                       .ToList();  // 关键点: resource.value
    }

    /**
     * 场景32: Stream filter 中的链式使用
     * 对应: list.stream().filter(obj -> obj.method() > 0)
     */
    public void testStreamFilterUse() {
        List<Resource> resources = new ArrayList<>();
        List<Resource> filtered = resources.stream()
                                         .filter(resource -> resource.value > 0)
                                         .filter(resource -> resource.parent != null)
                                         .ToList();  // 关键点: resource.value + resource.parent
    }

    // ==================== Optional 操作中的使用场景 ====================

    /**
     * 场景33: Optional map 中的链式使用
     * 对应: Optional.of(obj).map(value -> value.method())
     */
    public void testOptionalMapUse() {
        Resource resource = new Resource();
        Optional<String> name = Optional.ofNullable(resource)
                                      .map(Resource::getName);  // 关键点: Resource::getName

        Optional<Integer> value = Optional.ofNullable(resource)
                                         .map(res -> res.value);  // 关键点: res.value
    }

    /**
     * 场景34: Optional 链式操作使用
     * 对应: Optional.of(obj).map().flatMap().orElse()
     */
    public void testOptionalChainUse() {
        Resource resource = new Resource();
        Optional<String> parentName = Optional.ofNullable(resource)
                                           .map(Resource::getParent)
                                           .map(Resource::getName)
                                           .orElse("default");  // 关键点: Optional 链式操作

        Optional<Integer> parentValue = Optional.ofNullable(resource)
                                             .map(Resource::getParent)
                                             .map(res -> res.value)
                                             .orElse(0);  // 关键点: 获取父级值
    }

    // ==================== 循环中的使用场景 ====================

    /**
     * 场景35: for 循环中的变量使用
     * 对应: for (int i = 0; i < var.length; i++)
     */
    public void testForLoopVariableUse() {
        Resource[] resources = new Resource[10];
        for (int i = 0; i < resources.length; i++) {
            System.out.println(resources[i].name);  // 关键点: resources[i].name
        }
    }

    /**
     * 场景36: 增强 for 循环中的使用
     * 对应: for (Resource res : resources)  { res.method() }
     */
    public void testEnhancedForLoopUse() {
        List<Resource> resources = new ArrayList<>();
        for (Resource resource : resources) {
            System.out.println(resource.name);  // 关键点: resource.name

            processResource(resource);  // 关键点: resource
        }
    }

    // ==================== 复杂嵌套使用场景 ====================

    /**
     * 场景37: 极度复杂的嵌套使用
     * 对应: 涵盖所有类型的复杂USE关系
     */
    public void testExtremelyComplexUse() {
        Resource resource = new Resource();
        String deepName = resource.getChildren()
                                 .get(0)
                                 .getParent()
                                 .getResource("key")
                                 .name;  // 关键点: 极度复杂的链式使用

        int deepValue = resource.getParent()
                              .getChildren()
                              .get(1)
                              .value;  // 关键点: 深层嵌套的值使用

        // 在复杂表达式中使用
        int calculated = (resource.value * 2) + (resource.getParent().value / 2);  // 关键点: 复杂表达式
    }

    /**
     * 场景38: 静态方法中的变量使用
     * 对应: 静态上下文中的实例字段使用
     */
    public static void testStaticContextUse(UseExpressionSegmenterCase instance) {
        Resource resource = instance.resource;
        String name = resource.name;  // 关键点: 静态方法中使用实例字段

        int value = instance.resource.value;  // 关键点: 静态方法中使用嵌套字段
    }

    // ==================== 辅助方法 ====================

    private void processResource(Resource resource) {
        System.out.println("Processing: " + resource.name);
    }

    private String formatResource(String name, String pattern) {
        return String.format(pattern, name);
    }

    private int calculateValue(int a, int b) {
        return a + b;
    }

    // ==================== 字段定义 ====================

    private String name = "default_name";
    private int value = 100;
    private Resource resource = new Resource();
}