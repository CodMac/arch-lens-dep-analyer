/* 展示层字段：类型全形态（多层嵌套 / union / enum class / 模板嵌套 / 匿名承载）。 */

namespace ty {

/* 三层嵌套 struct */
struct L1 {
    struct L2 {
        struct L3 {
            int deep;
        };
        int mid;
    };
    int top;
};

/* union */
union U {
    int ui;
    float uf;
};

/* enum class */
enum class ScopedE {
    A,
    B
};

/* 普通 enum */
enum PlainE {
    P_A,
    P_B
};

/* 模板类内嵌套 */
template <typename T>
struct Box {
    T value;
    struct Nested {
        T n;
    };
    T get() { return value; }
};

/* 模板特化：QN 剥离实参 */
template <>
struct Box<int> {
    int special;
};

} // namespace ty

/* 匿名 struct 承载：typedef 名（TypeQN 为空，typedef 不是类型 kind） */
typedef struct {
    int tx;
    int ty;
} AnonPoint;

/* 匿名 struct 承载：变量名（TypeQN 为空） */
struct {
    int vx;
} anon_obj;

/* 匿名 union 承载：字段名 */
struct WithUnion {
    union {
        int as_int;
        float as_float;
    } payload;
};

/* 匿名 struct 嵌套在具名 struct 的字段下 */
struct OuterHolder {
    struct {
        int pair_a;
        int pair_b;
    } pair;
};
