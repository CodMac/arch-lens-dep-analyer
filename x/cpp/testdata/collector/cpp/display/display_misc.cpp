/* 展示层字段：杂项（全局变量 / macro / extern "C" / 多声明器 / = default / 运算符）。 */

#define DISP_MACRO 1
#define DISP_FUNC_MACRO(x) ((x) + 1)

namespace misc {

int g_var = 0;
static int g_static = 1;

/* 多声明器 */
int multi_a, multi_b, *multi_ptr;

/* = default / = delete */
struct Defaulted {
    Defaulted() = default;
    ~Defaulted() = default;
    Defaulted(const Defaulted &) = delete;
    int value;
};

/* 运算符成员 */
struct Operated {
    int v;
    Operated &operator+=(int x) {
        v += x;
        return *this;
    }
    int operator()(int x) const { return v + x; }
    explicit operator bool() const { return v != 0; }
};

/* extern "C" 内的声明 */
extern "C" {
int c_linkage_fn(int);
int c_linkage_var;
}

} // namespace misc
