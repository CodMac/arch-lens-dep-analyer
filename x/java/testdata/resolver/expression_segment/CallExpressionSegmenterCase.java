package com.example.resolver.segmenter.call;

import java.util.List;
import java.util.ArrayList;
import java.util.stream.Collectors;

/**
 * ExpressionSegmenter CALL 关系类型全场景测试用例
 * 专门测试方法调用（method_invocation）相关的链式表达式解析
 * 覆盖 expression_segmenter.go 中的 method_invocation 分支处理
 */
public class CallExpressionSegmenterCase {

    // ==================== 辅助类定义 ====================

    public static class Service {
        public String process(String input) {
            return input.toUpperCase();
        }

        public Service transform(String input) {
            return this;
        }

        public Service filter(String condition) {
            return this;
        }

        public String execute() {
            return "executed";
        }

        public static Service create() {
            return new Service();
        }

        public static String staticProcess(String input) {
            return input.trim();
        }

        public Data getData() {
            return new Data();
        }

        public List<Data> getDataList() {
            return new ArrayList<>();
        }

        public Data getElement(int index) {
            return new Data();
        }
    }

    public static class Data {
        public String value;
        public Data nested;

        public String getValue() {
            return value;
        }

        public Data getNested() {
            return nested;
        }

        public String extract(String prefix) {
            return prefix + value;
        }

        public Data append(String suffix) {
            value += suffix;
            return this;
        }
    }

    // ==================== 简单方法调用场景 ====================

    /**
     * 场景1: 简单无参方法调用
     * 对应: obj.method()
     */
    public void testSimpleMethodCall() {
        Service service = new Service();
        String result = service.process();  // 关键点: service.process()
    }

    /**
     * 场景2: 带参数的方法调用
     * 对应: obj.method(param)
     */
    public void testMethodCallWithParam() {
        Service service = new Service();
        String result = service.process("test");  // 关键点: service.process("test")
    }

    /**
     * 场景3: 多参数方法调用
     * 对应: obj.method(param1, param2)
     */
    public void testMethodCallWithMultipleParams() {
        Service service = new Service();
        Data data = new Data();
        String result = data.extract("prefix", "param2");  // 关键点: data.extract("prefix", "param2")
    }

    // ==================== 连续方法调用场景 ====================

    /**
     * 场景4: 两层连续方法调用
     * 对应: obj.method1().method2()
     */
    public void testTwoMethodChain() {
        Service service = new Service();
        String result = service.process("test").toUpperCase();  // 关键点: service.process("test").toUpperCase()
    }

    /**
     * 场景5: 三层连续方法调用
     * 对应: obj.method1().method2().method3()
     */
    public void testThreeMethodChain() {
        Service service = new Service();
        String result = service.transform("test").filter("cond").execute();  // 关键点: service.transform("test").filter("cond").execute()
    }

    /**
     * 场景6: 多层连续方法调用（深层）
     * 对应: obj.method1().method2().method3().method4().method5()
     */
    public void testDeepMethodChain() {
        Service service = new Service();
        String result = service.transform("input")
                           .filter("cond")
                           .transform("input2")
                           .filter("cond2")
                           .execute();  // 关键点: 长链式调用
    }

    // ==================== 字段访问+方法调用场景 ====================

    /**
     * 场景7: 字段访问后方法调用
     * 对应: obj.field1.method()
     */
    public void testFieldThenMethodCall() {
        Data data = new Data();
        String result = data.value.toLowerCase();  // 关键点: data.value.toLowerCase()

        Service service = new Service();
        Data nested = service.getData().getNested();  // 关键点: service.getData().getNested()
    }

    /**
     * 场景8: 方法调用后字段访问再方法调用
     * 对应: obj.method1().field2.method2()
     */
    public void testMethodThenFieldThenMethod() {
        Service service = new Service();
        String result = service.getData().value.toLowerCase();  // 关键点: service.getData().value.toLowerCase()
    }

    /**
     * 场景9: 混合字段和方法访问
     * 对应: obj.field1.method1().field2.method2()
     */
    public void testMixedFieldAndMethodAccess() {
        Service service = new Service();
        String result = service.getData().value.trim().toUpperCase();  // 关键点: service.getData().value.trim().toUpperCase()
    }

    // ==================== 数组/集合+方法调用场景 ====================

    /**
     * 场景10: 数组访问后方法调用
     * 对应: obj.array[0].method()
     */
    public void testArrayAccessThenMethodCall() {
        Service service = new Service();
        Data[] dataArray = new Data[10];
        String result = dataArray[0].toLowerCase();  // 关键点: dataArray[0].toLowerCase()
    }

    /**
     * 场景11: 集合方法调用后元素访问
     * 对应: obj.method().get(0).method()
     */
    public void testCollectionMethodThenElementMethod() {
        Service service = new Service();
        String result = service.getDataList().get(0).getValue();  // 关键点: service.getDataList().get(0).getValue()
    }

    /**
     * 场景12: 获取数组元素后方法调用
     * 对应: obj.method(index).method()
     */
    public void testGetElementThenMethodCall() {
        Service service = new Service();
        Data data = service.getElement(0);
        String result = data.getValue().toUpperCase();  // 关键点: data.getValue().toUpperCase()
    }

    // ==================== 静态方法调用场景 ====================

    /**
     * 场景13: 静态方法调用
     * 对应: ClassName.staticMethod()
     */
    public void testStaticMethodCall() {
        String result = Service.staticProcess("  test  ");  // 关键点: Service.staticProcess("  test  ")
    }

    /**
     * 场景14: 静态方法调用后继续方法调用
     * 对应: ClassName.staticMethod().method()
     */
    public void testStaticMethodThenInstanceMethod() {
        String result = Service.create().process("test");  // 关键点: Service.create().process("test")

        Service service = Service.staticProcess("input").trim();  // 编译错误，仅为示例
    }

    /**
     * 场景15: 系统静态方法调用（标准库）
     * 对应: System.out.println(), String.valueOf()
     */
    public void testLibraryStaticMethodCalls() {
        String result = String.valueOf(123).toLowerCase();  // 关键点: String.valueOf(123).toLowerCase()

        System.out.println("test".toUpperCase());  // 关键点: System.out.println()
    }

    // ==================== this/super 方法调用场景 ====================

    /**
     * 场景16: this 关键字方法调用
     * 对应: this.method()
     */
    public void testThisMethodCall() {
        String result = this.processInternal("test");  // 关键点: this.processInternal("test")
    }

    /**
     * 场景17: this 关键字字段访问后方法调用
     * 对应: this.field.method()
     */
    public void testThisFieldThenMethodCall() {
        String result = this.data.value.toUpperCase();  // 关键点: this.data.value.toUpperCase()
    }

    /**
     * 场景18: super 关键字方法调用
     * 对应: super.method()
     */
    public void testSuperMethodCall() {
        String result = super.toString();  // 关键点: super.toString()
    }

    // ==================== 括号内方法调用场景 ====================

    /**
     * 场景19: 括号包裹的方法调用
     * 对应: (obj.method()).otherMethod()
     */
    public void testParenthesizedMethodCall() {
        Service service = new Service();
        String result = (service.process("test")).toUpperCase();  // 关键点: (service.process("test")).toUpperCase()
    }

    /**
     * 场景20: 多层括号包裹的方法调用
     * 对应: ((obj.method())).otherMethod()
     */
    public void testNestedParenthesizedMethodCall() {
        Service service = new Service();
        String result = ((service.process("test"))).toLowerCase();  // 关键点: ((service.process("test"))).toLowerCase()
    }

    // ==================== 方法调用作为参数场景 ====================

    /**
     * 场景21: 方法调用作为方法参数
     * 对应: obj.method1(obj.method2())
     */
    public void testMethodCallAsParameter() {
        Service service = new Service();
        Data data = new Data();
        String result = service.process(data.getValue());  // 关键点: data.getValue()
    }

    /**
     * 场景22: 链式方法调用作为参数
     * 对应: obj.method1(obj.method2().method3())
     */
    public void testChainedMethodCallAsParameter() {
        Service service = new Service();
        String result = service.process(service.getData().getValue());  // 关键点: service.getData().getValue()
    }

    // ==================== 构建器/流式API场景 ====================

    /**
     * 场景23: 构建器模式链式调用
     * 对应: obj.builder().setA().setB().build()
     */
    public void testBuilderPatternChain() {
        Service service = new Service();
        String result = service.transform("input")
                           .filter("cond")
                           .transform("input2")
                           .execute();  // 关键点: 构建器模式

        // 另一种构建器模式
        String result2 = new Service()
                           .transform("input")
                           .filter("cond")
                           .execute();  // 关键点: new 对象后直接构建
    }

    /**
     * 场景24: Stream API 链式调用
     * 对应: list.stream().map().filter().collect()
     */
    public void testStreamApiChain() {
        List<String> list = new ArrayList<>();
        String result = list.stream()
                           .map(String::toUpperCase)
                           .filter(s -> s.length() > 3)
                           .collect(Collectors.joining(","));  // 关键点: Stream API 链式调用

        // 获取流操作结果
        String first = list.stream().findFirst().orElse("");  // 关键点: Stream 方法链
    }

    /**
     * 场景25: Optional 链式调用
     * 对应: Optional.of().map().orElse()
     */
    public void testOptionalChain() {
        Service service = new Service();
        String result = Optional.ofNullable(service.getData())
                                .map(Data::getValue)
                                .orElse("");  // 关键点: Optional 链式调用

        // Optional 方法链
        String upper = Optional.of("test")
                             .map(String::toUpperCase)
                             .orElse("");  // 关键点: Optional 方法链
    }

    // ==================== 复杂嵌套场景 ====================

    /**
     * 场景26: 嵌套在表达式中的方法调用
     * 对应: obj.method() + other.method()
     */
    public void testMethodCallInExpression() {
        Service service = new Service();
        String result = service.process("a") + service.process("b");  // 关键点: 表达式中的方法调用
    }

    /**
     * 场景27: 条件表达式中的方法调用
     * 对应: condition ? obj.method1() : obj.method2()
     */
    public void testMethodCallInConditional() {
        Service service = new Service();
        String result = service.getData() != null
            ? service.getData().getValue()
            : "default";  // 关键点: 条件表达式中的方法调用
    }

    /**
     * 场景28: 极度复杂的混合调用链
     * 对应: 涵盖所有类型的复杂CALL关系
     */
    public void testExtremelyComplexCallChain() {
        Service service = new Service();
        String result = service.getData().getNested()
                           .getValue()
                           .toUpperCase()
                           .trim()
                           .concat("_suffix");  // 关键点: 极度复杂

        // 更复杂的嵌套
        String complex = service.getDataList()
                               .get(0)
                               .getNested()
                               .getValue()
                               .toLowerCase()
                               .replace("old", "new");  // 关键点: 多层嵌套
    }

    // ==================== Lambda/Stream 高级场景 ====================

    /**
     * 场景29: Lambda 表达式中的方法引用
     * 对应: stream().map(Obj::method)
     */
    public void testLambdaMethodReference() {
        List<Data> dataList = new ArrayList<>();
        List<String> resultList = dataList.stream()
                                         .map(Data::getValue)  // 关键点: 方法引用
                                         .collect(Collectors.toList());
    }

    /**
     * 场景30: Lambda 中的对象方法调用
     * 对应: list.forEach(obj -> obj.method())
     */
    public void testLambdaObjectMethodCall() {
        List<Data> dataList = new ArrayList<>();
        dataList.forEach(data -> data.getValue().toUpperCase());  // 关键点: Lambda 中的对象方法调用
    }

    // ==================== 辅助方法和字段 ====================

    private String processInternal(String input) {
        return input.trim();
    }

    private Data data = new Data();
    private Service service = new Service();
}