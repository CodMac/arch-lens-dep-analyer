/* Resolver 继承链成员查找样例。 */
#include "inherit.h"

namespace inh {

int Root::root_method() { return 1; }
int MidA::mid_a_method() { return 2; }
int MidB::mid_b_method() { return 3; }
int Leaf::leaf_method() { return 4; }
int Overrider::root_method() { return 5; }

void inherit_user(Leaf *leaf, Overrider *ov) {
    /* 直接成员 */
    int a = leaf->leaf_field;
    int b = leaf->leaf_method();

    /* 一级基类成员 */
    int c = leaf->mid_a_field;
    int d = leaf->mid_a_method();
    int e = leaf->mid_b_field;
    int f = leaf->mid_b_method();

    /* 二级基类成员（菱形：两条路径都能到 Root） */
    int g = leaf->root_field;
    int h = leaf->root_method();

    /* 覆盖：命中派生类而非基类 */
    int i = ov->root_method();
    (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
    (void)g; (void)h; (void)i;
}

} // namespace inh
