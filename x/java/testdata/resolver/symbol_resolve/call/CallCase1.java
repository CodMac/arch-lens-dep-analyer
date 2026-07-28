package com.test;

import java.util.List;
import java.util.function.Function;
import com.test.case1.ParentService;
import com.test.case1.UserService;
import com.test.case1.User;
import com.test.case1.Order;

public class CallCase1 extends ParentService {
    private UserService userService;
    private User currentUser;

    public void execute() {
        // 维度 1: 基础与跨类调用 (Basic & Cross-Class Call)

        // Line 18: 场景 1.1 - 同类隐式方法调用
        doInternal();

        // Line 21: 场景 1.2 - 实例属性方法调用
        userService.saveUser(currentUser);

        // Line 24: 场景 1.3 - 静态方法调用
        StringUtils.isEmpty("test");


        // 维度 2: 继承链与多态调用 (Inheritance & Keywords)

        // Line 30: 场景 2.1 - this 关键字调用
        this.doInternal();

        // Line 33: 场景 2.2 - super 父类方法调用
        super.parentExecute();

        // Line 36: 场景 2.3 - 接口/多态方法调用
        EventListener listener = getListener();
        listener.onEvent();


        // 维度 3: 方法重载决议 (Overload Resolution)

        // Line 43: 场景 3.1 - 基本类型精确重载匹配 (应当命中 process(int))
        process(100);

        // Line 46: 场景 3.2 - 多参数对象重载匹配 (应当命中 process(String, User))
        process("admin", currentUser);


        // 维度 4: 链式连续调用 (Chained Calls)

        // Line 52: 场景 4.1 - Builder 链式流转调用 (断言目标为 build())
        new UserBuilder().setName("Alice").build();

        // Line 55: 场景 4.2 - 跨类深度链式调用 (断言目标为 getAmount())
        currentUser.getOrders().get(0).getAmount();


        // 维度 5: 高阶函数与方法引用 (Method Reference & External)

        // Line 61: 场景 5.1 - 方法引用调用
        Function<User, String> nameGetter = User::getName;

        // Line 64: 场景 5.2 - 未导入外部工具类方法调用 (保底)
        UnimportedTool.runTask();
    }

    private void doInternal() {}
    private void process(int count) {}
    private void process(String name, User user) {}
    private EventListener getListener() { return null; }

    public interface EventListener {
        void onEvent();
    }

    public static class StringUtils {
        public static boolean isEmpty(String str) { return str == null || str.isEmpty(); }
    }

    public static class UserBuilder {
        public UserBuilder setName(String name) { return this; }
        public User build() { return new User(); }
    }
}