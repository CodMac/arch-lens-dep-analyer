/* Resolver lambda 变量调用样例。 */
#include "lambda_var.h"

namespace lam {

int lambda_caller(int seed) {
    int base = seed + 1;

    /* 变量持有 lambda：f() 应解析到 Lambda 元素 */
    auto f = [base](int v) { return v + base; };

    /* 局部变量 lambda 与全局函数同名不冲突 */
    int r = f(2);

    /* 直接立即调用 lambda */
    int s = [](int v) { return v * 2; }(3);

    return r + s;
}

} // namespace lam
