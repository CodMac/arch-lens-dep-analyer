/* Resolver 继承链成员查找样例：多级继承、多继承、菱形。 */
#ifndef RESOLVER_INHERIT_H
#define RESOLVER_INHERIT_H

namespace inh {

struct Root {
    int root_field;
    int root_method();
};

struct MidA : public Root {
    int mid_a_field;
    int mid_a_method();
};

struct MidB : public Root {
    int mid_b_field;
    int mid_b_method();
};

/* 多继承：两个分支各自继承 Root */
struct Leaf : public MidA, public MidB {
    int leaf_field;
    int leaf_method();
};

/* 覆盖：同名方法在派生类 */
struct Overrider : public Root {
    int root_method();
};

} // namespace inh

#endif // RESOLVER_INHERIT_H
