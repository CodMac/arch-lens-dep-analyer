/* typedef / using：基础、指针、数组、函数指针、多声明器、链式、别名模板。 */

/* 基础别名 */
typedef unsigned long ulong_t;
typedef int my_int;

/* 指针 / 引用别名 */
typedef int* int_ptr_t;
typedef int& int_ref_t;

/* 数组别名 */
typedef int int_arr_t[10];

/* 函数指针别名 */
typedef int (*callback_t)(int, int);
typedef void (*void_cb_t)();

/* 多声明器 */
typedef int TI1, TI2, *PTI3;

/* 链式别名 */
typedef ulong_t my_ulong_t;

/* using 别名 */
using UInt = unsigned int;
using UIntPtr = UInt*;
using Callback = int (*)(int);

/* 别名模板 */
template <typename T>
using Vec = T*;

/* 具名 struct + 别名：结构体与别名各自独立 */
struct Named {
    int n;
};
typedef Named Named_t;
using NamedAlias = Named;

/* 匿名 struct typedef：字段挂 typedef 名 */
typedef struct {
    int x;
    int y;
} Point;

/* 别名使用 */
ulong_t g_ulong;
my_int g_myint;
int_ptr_t g_ptr;
callback_t g_cb;
UInt g_uint;
