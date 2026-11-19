/* Resolver 形态样例：段 1 归一化的 7 种 exprForm。 */
#ifndef RESOLVER_FORMS_H
#define RESOLVER_FORMS_H

namespace frm {

struct TypeX {
    int field;
    int method();
    static int static_method();
};

namespace sub {
int sub_func();
}

int free_func(int a);

} // namespace frm

#endif // RESOLVER_FORMS_H
