/* 函数：自由函数、可变参数、noexcept、constexpr、尾置返回、默认实参、重载、static/inline、extern "C"。 */

/* 无参 / void 参数 */
void no_args() {}
void void_args(void) {}

/* 可变参数 */
int varargs(const char* fmt, ...) {
    return 0;
}

/* noexcept */
void noex() noexcept {}
void noex_cond() noexcept(true) {}

/* constexpr / consteval */
constexpr int cx(int v) {
    return v * 2;
}

/* 尾置返回类型 */
auto trailing(int v) -> int {
    return v;
}

/* 默认实参 */
int with_defaults(int a, int b = 2, int c = 3) {
    return a + b + c;
}

/* 重载组 */
int overloaded(int a) {
    return a;
}
int overloaded(int a, int b) {
    return a + b;
}
double overloaded(double a) {
    return a;
}

/* static / inline */
static int static_fn() {
    return 0;
}
inline int inline_fn() {
    return 1;
}

/* extern "C" */
extern "C" {
void c_linkage_fn(void);
int c_linkage_var;
}

/* 返回指针 / 引用 */
int* ptr_ret() {
    return nullptr;
}
int& ref_ret() {
    static int v = 0;
    return v;
}

/* 递归 */
int factorial(int n) {
    return n <= 1 ? 1 : n * factorial(n - 1);
}
