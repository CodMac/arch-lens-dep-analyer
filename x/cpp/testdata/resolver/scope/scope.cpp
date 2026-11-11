/* Resolver 作用域链样例：短名沿词法作用域上溯、相对限定名补全、块内遮蔽。 */
#include "scope.h"

namespace outer {

namespace inner {

int scoped_func() { return 1; }

} // namespace inner

int outer_func() { return 2; }

int OuterType::NestedType::nested_method() { return 1; }

/* 短名上溯：inner::scoped_func 由相对限定名解析 */
int use_scoped() {
    return inner::scoped_func() + outer_func();
}

/* 嵌套类成员：NestedType 在 OuterType 内 */
int use_nested(OuterType::NestedType *n) {
    return n->nested_field + n->nested_method();
}

/* 块作用域遮蔽：内层局部变量优先 */
int shadowing(int v) {
    int result = 0;
    {
        int v = 10;
        result = v;
    }
    result += v;
    return result;
}

} // namespace outer
