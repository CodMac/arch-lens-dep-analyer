/* Lambda：无捕获/值捕获/引用捕获/全捕获、generic、mutable、作变量/字段/实参/返回值。 */

struct Holder {
    /* Lambda 作字段初始值 */
    int (*fn_field)(int) = [](int x) { return x; };
};

void lambda_user() {
    int outer = 1;
    int other = 2;

    /* 无捕获 */
    auto no_cap = [](int x) { return x + 1; };

    /* 值捕获 */
    auto by_val = [outer](int x) { return x + outer; };

    /* 引用捕获 */
    auto by_ref = [&outer](int x) { outer += x; return outer; };

    /* 全捕获 */
    auto all_val = [=](int x) { return x + outer + other; };
    auto all_ref = [&](int x) { return x + outer + other; };

    /* mutable */
    auto mut = [outer](int x) mutable { outer += x; return outer; };

    /* generic lambda */
    auto generic = [](auto a, auto b) { return a + b; };
    auto generic_pack = [](auto&&... xs) { return sizeof...(xs); };

    /* 嵌套 lambda */
    auto outer_lam = [outer]() {
        auto inner_lam = [outer]() { return outer; };
        return inner_lam();
    };

    /* 作实参 */
    auto call_with = [](int (*cb)(int), int v) { return cb(v); };

    /* 立即调用 */
    int immediate = [](int x) { return x * 2; }(5);

    no_cap(1);
    by_val(1);
    by_ref(1);
    all_val(1);
    all_ref(1);
    mut(1);
    generic(1, 2);
    generic_pack(1, 2, 3);
    outer_lam();
    call_with(no_cap, 1);
}

/* Lambda 作返回值 */
auto make_adder(int base) {
    return [base](int x) { return x + base; };
}
