/* Resolver 作用域链样例：嵌套 namespace、嵌套类、块作用域、同名遮蔽。 */
#ifndef RESOLVER_SCOPE_H
#define RESOLVER_SCOPE_H

namespace outer {
namespace inner {

int scoped_func();

struct InnerType {
    int inner_field;
};

} // namespace inner

int outer_func();

struct OuterType {
    struct NestedType {
        int nested_field;
        int nested_method();
    };

    int outer_field;
    NestedType nested;
};

} // namespace outer

#endif // RESOLVER_SCOPE_H
