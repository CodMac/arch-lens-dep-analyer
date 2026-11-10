/* struct/union：具名、嵌套、位域、匿名成员、匿名多声明器、union、自引用。 */

struct Node {
    int value;
    struct Node *next;
};

struct Outer {
    int id;
    struct Inner {
        int depth;
    } inner;
    union {
        int as_int;
        float as_float;
    } payload;
};

struct Bits {
    unsigned int flag : 1;
    unsigned int count : 3;
};

/* 匿名 struct 多声明器：每个变量各挂一份字段 */
struct {
    int m1;
    int m2;
} s1, s2;

/* union 具名 */
union Value {
    int i;
    double d;
};

/* union 匿名多声明器 */
union {
    int u1;
    long u2;
} MU1, MU2;
