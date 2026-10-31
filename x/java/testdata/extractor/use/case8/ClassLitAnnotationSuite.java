package com.example.audit;

import java.io.File;

// 注解类型元素默认值：default File.class（class_literal 位于 annotation_type_element_declaration）
@interface ClassLitMarker {
    Class<?> value() default File.class;
}

// 类级注解：@ClassLitMarker(value = File.class)（class_literal 位于类型声明的 modifiers 内）
@ClassLitMarker(value = File.class)
public class ClassLitAnnotationSuite {

    // 枚举构造器实参：FILE(File.class)（class_literal 位于 enum_constant 的实参列表）
    public enum Kind {
        FILE(File.class);

        private final Class<?> type;

        Kind(Class<?> type) {
            this.type = type;
        }
    }
}
