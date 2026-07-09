package com.example.resolver.segmenter.assign;

import java.util.List;
import java.util.ArrayList;
import java.util.Map;
import java.util.HashMap;

/**
 * ExpressionSegmenter ASSIGN 关系类型全场景测试用例
 * 专门测试赋值操作相关的链式表达式解析
 * 覆盖 expression_segmenter.go 中的赋值左值和右值的表达式解析
 */
public class AssignExpressionSegmenterCase {

    // ==================== 辅助类定义 ====================

    public static class Entity {
        public String name;
        public Entity parent;
        public Entity child;
        public List<Entity> children;
        public Entity[] entities;
        public Map<String, Entity> entityMap;

        public Entity getParent() {
            return parent;
        }

        public Entity getChild() {
            return child;
        }

        public List<Entity> getChildren() {
            return children;
        }

        public Entity getEntity(int index) {
            return entities[index];
        }

        public Entity getEntity(String key) {
            return entityMap.get(key);
        }
    }

    public static class Container {
        public Entity root;
        public Entity current;
        private Entity internal;

        public Entity getRoot() {
            return root;
        }

        public Entity getCurrent() {
            return current;
        }

        public Entity getInternal() {
            return internal;
        }

        public Entity findEntity(String id) {
            return root;
        }
    }

    public static class Builder {
        public String name;
        public Builder parent;
        public List<Builder> builders;

        public Builder setName(String name) {
            this.name = name;
            return this;
        }

        public Builder setParent(Builder parent) {
            this.parent = parent;
            return this;
        }
    }

    // ==================== 简单变量赋值场景 ====================

    /**
     * 场景1: 简单局部变量赋值
     * 对应: localVar = value
     */
    public void testSimpleVariableAssignment() {
        String localVar = "value";  // 关键点: localVar = "value"
        int count = 100;            // 关键点: count = 100
        boolean flag = true;        // 关键点: flag = true
    }

    /**
     * 场景2: 对象引用赋值
     * 对应: objRef = newObject
     */
    public void testObjectAssignment() {
        Entity entity = new Entity();  // 关键点: entity = new Entity()
        Container container = new Container();  // 关键点: container = new Container()
    }

    /**
     * 场景3: 方法返回值赋值
     * 对应: var = obj.method()
     */
    public void testMethodReturnValueAssignment() {
        Container container = new Container();
        Entity entity = container.getRoot();  // 关键点: entity = container.getRoot()

        String name = entity.name;  // 关键点: name = entity.name
    }

    // ==================== 字段赋值场景 ====================

    /**
     * 场景4: 简单字段赋值
     * 对应: obj.field = value
     */
    public void testSimpleFieldAssignment() {
        Entity entity = new Entity();
        entity.name = "entity1";  // 关键点: entity.name = "entity1"

        Container container = new Container();
        container.root = entity;  // 关键点: container.root = entity
    }

    /**
     * 场景5: 嵌套字段赋值
     * 对应: obj.field1.field2 = value
     */
    public void testNestedFieldAssignment() {
        Entity entity = new Entity();
        entity.parent.name = "parent1";  // 关键点: entity.parent.name = "parent1"

        Container container = new Container();
        container.root.parent.name = "root_parent";  // 关键点: container.root.parent.name = "root_parent"
    }

    /**
     * 场景6: 深层嵌套字段赋值
     * 对应: obj.field1.field2.field3.field4 = value
     */
    public void testDeepNestedFieldAssignment() {
        Entity entity = new Entity();
        entity.parent.child.name = "deep_entity";  // 关键点: entity.parent.child.name = "deep_entity"

        Container container = new Container();
        container.root.current.root.name = "very_deep";  // 关键点: container.root.current.root.name = "very_deep"
    }

    // ==================== this 字段赋值场景 ====================

    /**
     * 场景7: this 字段赋值
     * 对应: this.field = value
     */
    public void testThisFieldAssignment() {
        this.name = "this_entity";  // 关键点: this.name = "this_entity"

        this.root = new Entity();  // 关键点: this.root = new Entity()
    }

    /**
     * 场景8: this 嵌套字段赋值
     * 对应: this.field1.field2 = value
     */
    public void testThisNestedFieldAssignment() {
        Entity entity = new Entity();
        this.root.name = "this_root";  // 关键点: this.root.name = "this_root"

        this.root.parent.name = "this_root_parent";  // 关键点: this.root.parent.name = "this_root_parent"
    }

    // ==================== 方法调用结果赋值场景 ====================

    /**
     * 场景9: 方法返回值赋值给变量
     * 对应: var = obj.method()
     */
    public void testMethodResultVariableAssignment() {
        Container container = new Container();
        Entity entity = container.getRoot();  // 关键点: entity = container.getRoot()

        String rootName = container.getRoot().name;  // 关键点: rootName = container.getRoot().name
    }

    /**
     * 场景10: 链式方法调用结果赋值
     * 对应: var = obj.method1().method2()
     */
    public void testChainedMethodResultAssignment() {
        Container container = new Container();
        Entity parent = container.getRoot().getParent();  // 关键点: parent = container.getRoot().getParent()

        String childName = container.getRoot().getChild().name;  // 关键点: childName = container.getRoot().getChild().name
    }

    // ==================== 数组元素赋值场景 ====================

    /**
     * 场景11: 数组简单元素赋值
     * 对应: obj.array[0] = value
     */
    public void testArraySimpleElementAssignment() {
        Entity[] entities = new Entity[10];
        entities[0] = new Entity();  // 关键点: entities[0] = new Entity()

        String[] names = new String[5];
        names[0] = "entity1";  // 关键点: names[0] = "entity1"
    }

    /**
     * 场景12: 对象数组的字段赋值
     * 对应: obj.array[0].field = value
     */
    public void testArrayElementFieldAssignment() {
        Entity[] entities = new Entity[10];
        entities[1] = new Entity();
        entities[1].name = "entity2";  // 关键点: entities[1].name = "entity2"
    }

    /**
     * 场景13: 嵌套数组元素字段赋值
     * 对应: obj.array[0].array1[1].field = value
     */
    public void testNestedArrayElementFieldAssignment() {
        Entity[] entities = new Entity[10];
        entities[2] = new Entity();
        entities[2].parent = new Entity();
        entities[2].parent.name = "nested_entity";  // 关键点: entities[2].parent.name
    }

    /**
     * 场景14: 方法返回数组元素赋值
     * 对应: obj.method()[0].field = value
     */
    public void testMethodReturnArrayElementAssignment() {
        Entity entity = new Entity();
        Entity child = entity.getChildren().get(0);  // 关键点: child = entity.getChildren().get(0)

        Collections singletonList = new ArrayList();
        Entity firstChild = entity.getChildren().get(0);
        firstChild.name = "first";  // 关键点: firstChild.name = "first"
    }

    // ==================== 集合元素赋值场景 ====================

    /**
     * 场景15: 集合方法返回元素赋值
     * 对应: var = list.method(0)
     */
    public void testCollectionMethodElementAssignment() {
        Entity entity = new Entity();
        List<Entity> children = new ArrayList<>();
        children.add(new Entity());

        Entity firstChild = children.get(0);  // 关键点: firstChild = children.get(0)
        firstChild.name = "first_child";  // 关键点: firstChild.name = "first_child"
    }

    /**
     * 场景16: 集合链式调用元素赋值
     * 对应: var = obj.method1().method2().get(0)
     */
    public void testChainedCollectionElementAssignment() {
        Container container = new Container();
        Entity entity = container.getRoot();
        Entity firstChild = entity.getChildren().get(0);  // 关键点: firstChild = entity.getChildren().get(0)

        String firstChildName = entity.getChildren().get(0).name;  // 关键点: firstChildName = entity.getChildren().get(0).name
    }

    // ==================== Map 操作赋值场景 ====================

    /**
     * 场景17: Map get 结果赋值
     * 对应: var = map.get(key)
     */
    public void testMapGetAssignment() {
        Entity entity = new Entity();
        Entity retrieved = entity.getEntity("key1");  // 关键点: retrieved = entity.getEntity("key1")

        String name = entity.getEntity("key2").name;  // 关键点: name = entity.getEntity("key2").name
    }

    /**
     * 场景18: Map 链式调用赋值
     * 对应: var = obj.method().get(key).field
     */
    public void testMapChainedAssignment() {
        Container container = new Container();
        Entity entity = container.findEntity("id1");
        Entity nested = entity.getEntity("nested_key");  // 关键点: nested = entity.getEntity("nested_key")

        String nestedName = entity.getEntity("nested_key").name;  // 关键点: nestedName = entity.getEntity("nested_key").name
    }

    // ==================== 复杂表达式赋值场景 ====================

    /**
     * 场景19: 链式访问结果赋值
     * 对应: var = obj.field1.method1().field2.method2()
     */
    public void testComplexChainAssignment() {
        Container container = new Container();
        String rootName = container.getRoot().name;  // 关键点: rootName = container.getRoot().name

        Entity parentEntity = container.getRoot().getParent();  // 关键点: parentEntity = container.getRoot().getParent()
        String parentName = parentEntity.name;  // 关键点: parentName = parentEntity.name
    }

    /**
     * 场景20: 方法参数链式访问赋值
     * 对应: var = obj.method(obj2.method2())
     */
    public void testMethodParameterChainAssignment() {
        Container container = new Container();
        Entity parent = container.getRoot().getParent();  // 关键点: parent = container.getRoot().getParent()

        String parentKey = parent.name;  // 关键点: parentKey = parent.name
    }

    // ==================== 括号内赋值场景 ====================

    /**
     * 场景21: 括号内字段访问赋值
     * 对应: (var = obj.field)
     */
    public void testParenthesizedAssignment() {
        Entity entity = new Entity();
        (entity.name) = "parenthesized";  // 关键点: (entity.name)

        Container container = new Container();
        (container.root.name) = "container_root";  // 关键点: (container.root.name)
    }

    /**
     * 场景22: 多层括号嵌套赋值
     * 对应: ((var = obj.field))
     */
    public void testNestedParenthesizedAssignment() {
        Entity entity = new Entity();
        ((entity.name)) = "double_parenthesized";  // 关键点: ((entity.name))
    }

    // ==================== 构建器模式赋值场景 ====================

    /**
     * 场景23: Builder 链式调用赋值
     * 对应: var = obj.setA().setB().build()
     */
    public void testBuilderChainAssignment() {
        Builder builder = new Builder();
        builder.setName("builder1")
               .setParent(new Builder());  // 关键点: builder.setName().setParent()

        // 另一种形式
        Builder nestedBuilder = builder.parent.setName("nested");  // 关键点: builder.parent.setName()
    }

    // ==================== 条件表达式中的赋值场景 ====================

    /**
     * 场景24: 条件表达式后的赋值
     * 对应: var = condition ? obj1.method() : obj2.method()
     */
    public void testConditionalAssignment() {
        Container container = new Container();
        Entity entity = container.getRoot();

        Entity selected = entity.parent != null
            ? entity.parent
            : entity.child;  // 关键点: selected = 条件表达式结果

        String selectedName = selected.name;  // 关键点: selectedName = selected.name
    }

    // ==================== 静态字段赋值场景 ====================

    /**
     * 场景25: 静态字段赋值
     * 对应: ClassName.field = value
     */
    public void testStaticFieldAssignment() {
        // 假设有静态字段
        ClassName.staticField = "static_value";  // 关键点: ClassName.staticField
    }

    // ==================== 方法链赋值场景 ====================

    /**
     * 场景26: 方法链最终赋值
     * 对应: finalField = obj.method1().method2().method3().finalField
     */
    public void testMethodChainFinalAssignment() {
        Container container = new Container();
        String deepName = container.getRoot().getParent().name;  // 关键点: 深层方法链赋值

        Entity deepEntity = container.getRoot().getChild().getChild();  // 关键点: 深层对象链赋值
    }

    /**
     * 场景27: 复杂的混合链赋值
     * 对应: 涵盖各种访问类型的复杂赋值
     */
    public void testComplexMixedAssignment() {
        Container container = new Container();
        Entity entity = container.getRoot();

        // 集合访问 + 方法调用
        Entity firstChild = entity.getChildren().get(0);  // 关键点: firstChild = entity.getChildren().get(0)

        // 数组访问 + 字段访问
        Entity[] entities = new Entity[10];
        String firstEntityName = entities[0].name;  // 关键点: firstEntityName = entities[0].name

        // Map 访问 + 方法调用
        Entity mapped = entity.getEntity("key").getChild();  // 关键点: mapped = entity.getEntity("key").getChild()

        // 深层混合访问
        String veryDeep = container.getRoot().getChildren().get(0).getEntity("key").name;  // 关键点: veryDeep = 深层混合链
    }

    // ==================== 边界情况赋值场景 ====================

    /**
     * 场景28: null 值赋值
     * 对应: obj.field = null
     */
    public void testNullAssignment() {
        Entity entity = new Entity();
        entity.parent = null;  // 关键点: entity.parent = null

        Container container = new Container();
        container.root = null;  // 关键点: container.root = null
    }

    /**
     * 场景29: 链式调用中的null赋值
     * 对应: obj.method1().field = null
     */
    public void testChainedNullAssignment() {
        Container container = new Container();
        Entity entity = container.getRoot();
        entity.parent = null;  // 关键点: entity.parent = null
    }

    // ==================== 辅助方法和字段 ====================

    private String name = "default_name";
    private Entity root = new Entity();
    private Container container = new Container();
}