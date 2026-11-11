package com.example.display;

/**
 * 接口：常量 / 抽象方法 / 默认方法 / 静态方法 + 嵌套 class / interface / enum / annotation。
 * 所有嵌套类型及其成员的 TypeQN 应为接口自身。
 */
public interface DisplayInterface<T> {

    int CONST = 1;

    void run();

    default int def() {
        return CONST;
    }

    static <U> U identity(U u) {
        return u;
    }

    class Impl {
        int implField;

        void implMethod() {
        }
    }

    interface Sub {
        void subMethod();
    }

    enum Mode {
        A, B
    }

    @interface Marker {
        String value();
    }
}
