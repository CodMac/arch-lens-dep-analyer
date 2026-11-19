/* Capture 样例：值捕获 / 引用捕获 / 全捕获 / 嵌套 lambda / 无捕获 / 自身局部变量不捕获。 */

int capture_cases(int seed) {
    int by_value = seed + 1;
    int by_ref = seed + 2;
    int unused = 0;
    int assigned = 0;
    (void)unused;

    // 值捕获
    auto f1 = [by_value]() { return by_value; };

    // 引用捕获
    auto f2 = [&by_ref]() { by_ref += 1; return by_ref; };

    // 全捕获（隐式捕获外部变量）
    auto f3 = [&]() { return by_value + by_ref; };

    // 无捕获
    auto f4 = []() { return 0; };

    // lambda 自身参数/局部变量不构成捕获
    auto f5 = [by_value](int p) {
        int inner = p + 1;
        return inner + by_value;
    };

    // 嵌套 lambda：内层捕获外层 lambda 的局部变量（非函数局部变量）
    auto f6 = [by_value]() {
        int mid = by_value + 10;
        auto inner = [mid]() { return mid; };
        return inner();
    };

    // 捕获变量被赋值（ASSIGN 推导 CAPTURE）
    auto f7 = [&assigned]() { assigned = 1; };

    return f1() + f2() + f3() + f4() + f5(1) + f6() + f7();
}
