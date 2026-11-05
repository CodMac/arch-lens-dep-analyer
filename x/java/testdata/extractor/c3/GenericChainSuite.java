package com.example.rel.c3;

import java.util.function.Supplier;

// C-3 #95 修复断言：容器泛型字段 C<X> 上的链式调用（如 Supplier<X>.get().method()）。
// 根因：字段类型 Supplier<X> 的泛型实参 X 未传播到外部方法段 .get()，
// 后续段在 X 上解析失败，断链为 Supplier.get.method() 外部裸名（方法 QN 污染）。
// 修复后：.get() 外部方法携带 MethodReturnType=X（来自字段类型文本泛型实参），
// 后续段在 X 容器上继续解析：
//   - X 为内部类型（InternalTarget）-> 精确解析到内部方法；
//   - X 为外部类型（String）-> 用容器 QN 解析为外部方法（java.lang.String.length()）。
public class GenericChainSuite {

    // 内部目标类型：X 为同包内部类，后续段应精确解析到内部方法。
    static class InternalTarget {
        public String internalMethod() { return "ok"; }
    }

    // 外部容器字段：Supplier<InternalTarget>。
    private final Supplier<InternalTarget> internalSupplier = InternalTarget::new;

    public String useInternal() {
        // 链式：internalSupplier.get().internalMethod()
        // 修复前：Supplier.get.internalMethod()（外部裸名，方法 QN 污染）
        // 修复后：com.example.rel.c3.GenericChainSuite.InternalTarget.internalMethod()（内部方法）
        return internalSupplier.get().internalMethod();
    }

    // 外部容器字段：Supplier<String>（X 为外部类型，模拟 spring 中
    // Supplier<jakarta.validation.Validator> 的 Validator 未注册进符号库）。
    private final Supplier<String> externalSupplier = () -> "x";

    public int useExternal() {
        // 链式：externalSupplier.get().length()
        // 修复前：Supplier.get.length()（外部裸名）
        // 修复后：java.lang.String.length()（容器 QN）
        return externalSupplier.get().length();
    }
}
