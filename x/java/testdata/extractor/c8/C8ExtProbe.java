package com.example.rel;

// 模拟：当前类继承外部基类（不在符号表），字段从外部继承
class C8ExtProbe extends SomeExternalBase {

    // Form 3 真实现：this.field 来自外部继承
    Object f3() {
        return this.pageContext.getAttribute("x");
    }

    // Form 3 变体：this.field 是外部静态字段？
    int f4() {
        return SomeExternalBase.SIZE;
    }
}
