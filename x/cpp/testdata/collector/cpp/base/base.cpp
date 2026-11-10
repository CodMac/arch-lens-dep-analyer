/* 基准元素：源文件侧的定义、局部变量、Lambda、块作用域。 */
#include "base.h"

namespace base_ns {

int base_global = 0;

BaseClass::BaseClass() : member(0) {}

void BaseClass::method() {
    int local = member;
    if (local > 0) {
        int in_block = local;
    }
}

int base_func(int a) {
    int sum = a;
    return sum;
}

void base_lambda_user() {
    auto lam = [](int x) { return x; };
    lam(1);
}

} // namespace base_ns
