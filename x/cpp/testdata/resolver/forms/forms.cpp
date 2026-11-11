/* Resolver 形态样例：formIdent / formMember / formScoped / formCall / formNew / formCast。 */
#include "forms.h"

namespace frm {

int TypeX::method() { return field; }
int TypeX::static_method() { return 7; }

int free_func(int a) { return a; }

namespace sub {
int sub_func() { return 3; }
}

void forms_user(TypeX *p) {
    /* formIdent：裸标识符 */
    int local = 1;
    int a = local;

    /* formMember：成员访问（. 与 ->） */
    int b = p->field;
    int c = p->method();

    /* formScoped：限定访问 */
    int d = sub::sub_func();
    int e = TypeX::static_method();

    /* formCall：调用（自由函数） */
    int f = free_func(a);

    /* formNew：new 表达式 */
    TypeX *n = new TypeX();

    /* formCast：C 风格 / 显式转换 */
    TypeX *g = (TypeX *)p;
    TypeX *h = static_cast<TypeX *>(p);

    (void)b; (void)c; (void)d; (void)e; (void)f;
    (void)n; (void)g; (void)h;
}

} // namespace frm
