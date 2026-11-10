/* 边界与负例：重名消歧、匿名 struct/union 承载、类型引用不产元素、负例。 */

/* 重名普通元素：同名不同 kind 共存，同 kind 重名 $N 消歧 */
struct DupTag {
    int a;
};
class DupTag2 {
public:
    int b;
};
int dup_var;
int dup_var2;

namespace dup_ns {
int dup_var;
} // namespace dup_ns

/* 同名函数重载：按参数区分，不消歧 */
int dup_fn(int a) {
    return a;
}
int dup_fn(int a, int b) {
    return a + b;
}

/* 匿名 struct 承载：typedef 名 */
typedef struct {
    int tx;
    int ty;
} AnonPoint;

/* 匿名 struct 承载：变量名 */
struct {
    int vx;
} anon_obj;

/* 匿名 union 承载：字段名 */
struct WithAnonUnion {
    union {
        int as_int;
        float as_float;
    } payload;
};

/* 类型引用不产元素 */
struct RefTarget {
    int rv;
};
void use_ref() {
    struct RefTarget local;
    struct RefTarget* ptr;
}

/* 前向声明不产字段 */
struct FwdOnly;

/* 空 struct */
struct EmptyStruct {};

/* 空函数 */
void empty_fn() {}

/* 只声明不定义 */
int declared_only(int a);

/* 无名参数 */
void unnamed_params(int, char*);

/* 负例：宏体中的类型名不产元素 */
#define FAKE_CLASS FakeClass
