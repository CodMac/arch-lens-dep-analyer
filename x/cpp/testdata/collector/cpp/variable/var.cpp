/* 变量：全局/static/const/constexpr/thread_local/extern/alignas、auto、引用、指针、数组、结构化绑定。 */

/* 全局变量族 */
int g_counter;
static int g_static;
const int g_const = 1;
constexpr int g_constexpr = 2;
thread_local int g_tl;
extern int g_extern;
alignas(16) int g_aligned;

/* auto / 引用 / 指针 / 数组 */
auto g_auto = 1;
int& g_ref = g_counter;
int&& g_rref = 1;
int* g_ptr;
int** g_pptr;
const int* g_ptr_to_const;
int* const g_const_ptr = nullptr;
int g_arr[10];
int g_arr2[2][3];

/* 多声明器 */
int g_m1, g_m2, *g_m3;

/* 初始化形式 */
int g_brace{5};
int g_paren(6);

void var_user(int param) {
    /* 局部变量 */
    int local;
    static int local_static;
    const int local_const = 1;
    constexpr int local_constexpr = 2;
    thread_local int local_tl;

    /* auto 推导 */
    auto l_auto_int = 1;
    auto l_auto_ptr = &local;

    /* 引用与指针 */
    int& l_ref = local;
    int&& l_rref = 1;
    int* l_ptr = &local;
    int** l_pptr = &l_ptr;
    const int* l_cptr = &local;

    /* 数组 */
    int l_arr[5];
    int l_arr2[2][2];

    /* 多声明器 */
    int l_m1, l_m2, *l_m3;

    /* 结构化绑定（C++17） */
    int pair_like[2] = {1, 2};
    auto [sb_first, sb_second] = pair_like;

    /* 块作用域 */
    {
        int inner_block;
    }
}
