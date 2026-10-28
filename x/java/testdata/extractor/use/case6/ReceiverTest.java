package com.example.rel.use.case6;

public class ReceiverTest {
    public void test() {
        User user = new User();
        // Source: com.example.rel.use.case6.ReceiverTest.test() (METHOD)
        // Target: com.example.rel.use.case6.User.getName() (METHOD)
        // Mores: { "java.rel.ast_kind": "method_invocation" }
        user.getName().trim(); // [Case 11] 解析 getName() 时，能否识别出 user 是 User 类型？
    }
}
