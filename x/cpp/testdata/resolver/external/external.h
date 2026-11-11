/* Resolver 外部符号兜底样例：基本类型 / 模板变量 / 外部类型 / 外部方法。 */
#ifndef RESOLVER_EXTERNAL_H
#define RESOLVER_EXTERNAL_H

namespace ext {

/* 源码内类型 */
struct Local {
    int v;
};

/* 参数为外部类型 std::string */
int use_external(const std::string &s);

/* 返回外部类型 */
std::string make_string();

/* 模板参数 T：不是真实类型依赖 */
template <typename T>
T identity(T v);

} // namespace ext

#endif // RESOLVER_EXTERNAL_H
