package com.example.resolver;

import java.util.List;

// 目标：验证泛型元素类型传播（R4）。
// getOrders() 返回 List<Order>，数组下标消费应传播 Order 元素类型。
public class GenericPropagationSuite {

    public static class Order {
        String id;

        String getId() { return id; }
    }

    public static List<Order> getOrders() { return null; }

    public void testGenericElement() {
        // 类名静态调用：GenericPropagationSuite.getOrders() 作为 segment，传播 List<Order> -> Order。
        Order first = GenericPropagationSuite.getOrders()[0];

        // 隐式调用：getOrders() 作为 head（不触发传播，head 不经 resolveSegment）。
        Order second = getOrders()[0];
    }
}
