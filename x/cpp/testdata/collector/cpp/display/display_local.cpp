/* 展示层字段：局部类型（方法内 / 自由函数内 / 块内 / lambda 内 / 局部套局部）。 */

namespace lc {

struct Host {
    int hf;

    void methodWithLocal() {
        struct LocalInMethod {
            int x;
        };
        LocalInMethod l;

        class LocalClassInMethod {
        public:
            int y;
        };
        LocalClassInMethod c;

        enum LocalEnumInMethod {
            LE_A
        };
    }

    struct NestedInHost {
        int n;
    };
};

} // namespace lc

/* 自由函数内的局部类型：无具名宿主类型，TypeQN 应为空 */
void freeFnWithLocal() {
    struct LocalInFree {
        int z;
    };
    LocalInFree l;

    {
        struct LocalInBlock {
            int w;
        };
        LocalInBlock b;
    }
}

/* lambda 内的局部类型 */
void lambdaWithLocal() {
    auto f = []() {
        struct LocalInLambda {
            int v;
        };
        LocalInLambda x;
        return x.v;
    };
    f();
}

/* 局部类套局部类 */
void localInLocal() {
    struct OuterLocal {
        void deepen() {
            struct InnerLocal {
                int deep;
            };
            InnerLocal i;
        }
    };
    OuterLocal o;
    o.deepen();
}
