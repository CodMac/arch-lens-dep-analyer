package com.example.rel;

public interface IfaceConst {
    String PLAIN = "plain";            // 隐式 public static final
    String FACTORY_BEAN_PREFIX = "&";  // 隐式 public static final

    static String staticM() { return ""; }
    default String defM() { return ""; }
}

class ClassConst {
    static final String STATIC_FIELD = "s";  // 显式 static final
    public static String PUB_STATIC = "p";   // 显式 public static
}
