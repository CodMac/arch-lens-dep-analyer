package com.example.lombok;

import lombok.Data;

@Data
public class EdgeCases {
    // 基本类型
    private int primitiveInt;
    private long primitiveLong;
    private boolean primitiveBoolean;
    private double primitiveDouble;

    // 包装类型
    private Integer wrapperInt;
    private Long wrapperLong;
    private Boolean wrapperBoolean;
    private Double wrapperDouble;

    // 字符串
    private String stringField;

    // 数组
    private int[] intArray;
    private String[] stringArray;

    // 泛型集合（可能需要特殊处理）
    // private java.util.List<String> genericList;

    // 静态字段（不应该生成getter/setter）
    private static String staticField = "static";

    // transient字段
    private transient String transientField;
}

public class EdgeCasesUsageTest {
    public void testEdgeCases() {
        EdgeCases obj = new EdgeCases();

        // 测试基本类型的getter/setter
        obj.setPrimitiveInt(123);
        int i = obj.getPrimitiveInt();

        obj.setPrimitiveLong(456L);
        long l = obj.getPrimitiveLong();

        obj.setPrimitiveBoolean(true);
        boolean b = obj.getPrimitiveBoolean();

        obj.setPrimitiveDouble(3.14);
        double d = obj.getPrimitiveDouble();

        // 测试包装类型的getter/setter
        obj.setWrapperInt(789);
        Integer wi = obj.getWrapperInt();

        obj.setWrapperLong(101112L);
        Long wl = obj.getWrapperLong();

        obj.setWrapperBoolean(false);
        Boolean wb = obj.getWrapperBoolean();

        obj.setWrapperDouble(2.718);
        Double wd = obj.getWrapperDouble();

        // 测试字符串的getter/setter
        obj.setStringField("test");
        String s = obj.getStringField();

        // 测试数组的getter/setter
        obj.setIntArray(new int[]{1, 2, 3});
        int[] ia = obj.getIntArray();

        obj.setStringArray(new String[]{"a", "b", "c"});
        String[] sa = obj.getStringArray();

        // 测试transient字段
        obj.setTransientField("transient");
        String tf = obj.getTransientField();

        // 测试equals, hashCode, toString
        String str = obj.toString();
        int hash = obj.hashCode();
        EdgeCases another = new EdgeCases();
        boolean eq = obj.equals(another);
    }
}