package com.example.base.test;

import java.util.List;
import java.util.Map;

// 泛型类：多类型参数 + 多界约束
public class GenericEdge<K extends Number, V extends Comparable<V> & Cloneable> {
    // 泛型字段（嵌套泛型 + 无界通配符 + 上界通配符）
    private List<String> names;
    private Map<String, List<Integer>> index;
    private List<?> rawWildcard;
    private List<? extends Number> upperWildcard;
    private List<? super Integer> lowerWildcard;

    // 泛型方法：返回类型参数 + 通配符参数
    public K firstKey() {
        return null;
    }

    public <T> T identity(T value) {
        return value;
    }

    public void consume(List<? super V> sink) {
    }

    // 泛型构造器
    public <E> GenericEdge(E seed, List<K> keys) {
    }

    // 泛型方法多界
    public <E extends Number & Comparable<E>> int compareWith(E other) {
        return 0;
    }

    // 通配符嵌套：? extends List<? super T>
    public void nestedWildcard(List<? extends List<? super K>> deep) {
    }
}
