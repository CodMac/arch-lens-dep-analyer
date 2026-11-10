/* 参数形态：值/指针/多级指针/左值引用/右值引用/const/数组/函数指针/默认值/可变参数/无名。 */

struct PStruct {
    int v;
};

enum PEnum {
    PE_A
};

/* 值参数 */
void by_value(int a, double b, char c) {}

/* 指针 / 多级指针 */
void by_pointer(int* p, int** pp, char*** ppp) {}

/* 引用 / 右值引用 */
void by_ref(int& r, const int& cr, int&& rr) {}

/* const 各种位置 */
void by_const(const int a, const int* p, int* const cp, const int& r) {}

/* 数组参数（退化为指针语义） */
void by_array(int arr[], int arr2[10], int arr3[2][3]) {}

/* 函数指针参数 */
void by_funcptr(int (*cb)(int, int), void (*cb2)()) {}

/* 默认值 */
void by_default(int a, int b = 2, const char* s = "x") {}

/* 可变参数 */
void by_variadic(int first, ...) {}

/* 无名参数 */
void by_unnamed(int, double, int*) {}

/* struct / enum 类型参数 */
void by_user_type(PStruct s, PEnum e, PStruct* ps) {}

/* 模板参数 */
template <typename T>
void by_template(T v, T* pv, T& rv) {}
