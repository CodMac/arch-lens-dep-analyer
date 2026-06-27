package com.example.resolver.chain;

import java.util.List;
import java.util.ArrayList;

/**
 * 链式调用复杂场景测试用例
 * 涵盖各种混合链式调用模式
 */
public class ChainCallComplexCase {

    // ==================== 相关类定义 ====================

    public static class Inner1 {
        public String data;
        public Inner2 inner2;

        public String processData() {
            return data.toUpperCase();
        }

        public Inner2 getInner2() {
            return inner2;
        }
    }

    public static class Inner2 {
        public String value;
        public Inner3 inner3;

        public String transform() {
            return value.trim();
        }

        public Inner3 getInner3() {
            return inner3;
        }
    }

    public static class Inner3 {
        public String info;

        public String format() {
            return info.toLowerCase();
        }
    }

    public static class ComplexContainer {
        public Inner1 inner1;
        public List<Inner1> innerList;
        public String result;

        public Inner1 getInner1() {
            return inner1;
        }

        public List<Inner1> getInnerList() {
            return innerList;
        }

        public String getResult() {
            return result;
        }

        public ComplexContainer setResult(String result) {
            this.result = result;
            return this;
        }
    }

    // ==================== 链式调用测试场景 ====================

    /**
     * 场景1: obj1.obj2.method1() - 字段访问+方法调用
     */
    public void fieldAccessThenMethodCall() {
        ComplexContainer container = new ComplexContainer();
        container.inner1 = new Inner1();
        container.inner1.data = "hello";

        // obj1.obj2.method1() 模式
        String result = container.inner1.processData();
    }

    /**
     * 场景2: method1().method2().method3() - 连续方法调用
     */
    public void continuousMethodCalls() {
        ComplexContainer container = new ComplexContainer();
        container.inner1 = new Inner1();
        container.inner1.data = "  hello  ";
        container.inner1.inner2 = new Inner2();
        container.inner1.inner2.value = "world";
        container.inner1.inner2.inner3 = new Inner3();
        container.inner1.inner2.inner3.info = "JAVA";

        // method1().method2().method3() 深层链式调用
        String result = container.getInner1().processData()
                                    .concat(container.getInner1().getInner2().transform())
                                    .concat(container.getInner1().getInner2().getInner3().format());
    }

    /**
     * 场景3: obj1.method1().obj2.method2().method3() - 混合模式
     */
    public void mixedChainCalls() {
        ComplexContainer container = new ComplexContainer();
        container.setInner1(new Inner1());
        container.inner1.data = "test";
        container.inner1.inner2 = new Inner2();
        container.inner1.inner2.value = "data";
        container.inner1.inner2.inner3 = new Inner3();
        container.inner1.inner2.inner3.info = "INFO";

        // obj1.method1().obj2.method2().method3() 混合链式调用
        // container.method1().result + container.result.method2().method3()
        String concatResult = container.getInner1().processData()
                            + "."
                            + container.setResult("new value").getResult().toLowerCase();
    }

    /**
     * 场景4: method1().obj1.method2() - 方法返回后字段访问
     */
    public void methodThenFieldAccessThenMethod() {
        ComplexContainer container = new ComplexContainer();
        container.innerList = new ArrayList<>();
        container.innerList.add(new Inner1());
        container.innerList.get(0).data = "item";
        container.innerList.get(0).inner2 = new Inner2();
        container.innerList.get(0).inner2.value = "value";

        // method1().obj1.method2() 模式
        // getInnerList().get(0).inner2.transform()
        String result = container.getInnerList()
                               .get(0)
                               .inner2
                               .transform();
    }

    /**
     * 场景5: 复杂的Builder模式链式调用
     */
    public void complexBuilderPattern() {
        ComplexContainer container = new ComplexContainer();
        container.inner1 = new Inner1();
        container.inner1.data = "initial";
        container.inner1.inner2 = new Inner2();
        container.inner1.inner2.value = "secondary";

        // 多层链式调用与方法返回值处理
        String processed = container.getInner1().processData()
                                   .concat("_" + container.getInner1().getInner2().transform())
                                   .toUpperCase()
                                   .trim();
    }

    /**
     * 场景6: 静态方法与实例方法的混合链式调用
     */
    public void mixedStaticAndInstance() {
        ComplexContainer container = new ComplexContainer();
        container.inner1 = new Inner1();
        container.inner1.data = "static_test";

        // 静态方法开头，然后混合实例方法
        String result = String.valueOf(container.getInner1().processData().length());
    }

    /**
     * 场景7: 条件表达式中的链式调用
     */
    public void conditionalChainedCalls() {
        ComplexContainer container = new ComplexContainer();
        container.inner1 = new Inner1();
        container.inner1.data = "conditional";
        container.inner1.inner2 = new Inner2();
        container.inner1.inner2.value = "test";

        // 条件表达式中的链式调用
        String result = (container.inner1.data.length() > 5)
            ? container.getInner1().processData().toUpperCase()
            : container.getInner1().getInner2().transform().toLowerCase();
    }

    /**
     * 场景8: 多层次的字段访问与方法调用交错
     */
    public void deeplyNestedMixedAccess() {
        ComplexContainer container = new ComplexContainer();
        container.inner1 = new Inner1();
        container.inner1.data = "deep";
        container.inner1.inner2 = new Inner2();
        container.inner1.inner2.value = "nested";
        container.inner1.inner2.inner3 = new Inner3();
        container.inner1.inner2.inner3.info = "access";

        // 极致的深层混合访问
        String result = container.inner1.inner2.inner3.format().concat("_").concat(container.getInner1().getInner2().transform());
    }

    /**
     * 场景9: 集合操作中的链式调用
     */
    public void collectionChainedCalls() {
        ComplexContainer container = new ComplexContainer();
        container.innerList = new ArrayList<>();

        for (int i = 0; i < 3; i++) {
            Inner1 inner1 = new Inner1();
            inner1.data = "item" + i;
            inner1.inner2 = new Inner2();
            inner1.inner2.value = "value" + i;
            container.innerList.add(inner1);
        }

        // 集合链式调用
        String firstItem = container.getInnerList().get(1).inner2.transform();
        String secondProcess = container.getInnerList().get(2).processData();
    }

    /**
     * 场景10: 方法传参中的链式调用
     */
    public void parameterChainedCalls() {
        ComplexContainer container = new ComplexContainer();
        container.inner1 = new Inner1();
        container.inner1.data = "parameter";

        // 方法参数中的链式调用
        processResult(container.getInner1().processData().toUpperCase());

        // 嵌套方法调用中的链式
        nestedMethodCall(container.getInner1().getInner2().transform().trim());
    }

    // 辅助方法
    public void processResult(String input) {
        System.out.println("Processing: " + input);
    }

    public void nestedMethodCall(String input) {
        System.out.println("Nested: " + input);
    }

    // Getter/Setter辅助方法
    public void setInner1(Inner1 inner1) {
        this.inner1 = inner1;
    }
}