package com.example.resolver;

import java.util.List;

// 目标：验证重载打分（R5）。
// handle(String) / handle(int) / handle(String,List) / handle(Object...) 多重重载。
public class OverloadSuite {

    public void handle(String s) {}
    public void handle(int i) {}
    public void handle(String s, List<String> tags) {}
    public void handle(Object... args) {}

    public void testOverloads() {
        handle("text");            // String -> handle(String)
        handle(42);                 // int -> handle(int)
        handle("a", List.of());    // (String, List) -> handle(String,List)
        handle(1, 2, 3);            // (int,int,int) -> handle(Object...)
    }
}
