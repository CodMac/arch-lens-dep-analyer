/* Resolver 外部符号兜底样例。 */
#include "external.h"
#include <string>
#include <vector>

namespace ext {

int use_external(const std::string &s) {
    return (int)s.size();
}

std::string make_string() {
    return std::string();
}

template <typename T>
T identity(T v) { return v; }

/* 外部类型变量 + 外部方法调用 */
int external_user() {
    std::vector<int> vec;
    vec.push_back(1);
    std::string s = make_string();
    return (int)vec.size() + (int)s.size();
}

/* 模板变量 T 不应解析为外部类型 */
void template_user() {
    int x = identity(1);
    (void)x;
}

} // namespace ext
