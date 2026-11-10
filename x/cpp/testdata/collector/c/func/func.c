/* 函数族：无参/void 参数/可变参数/static/inline/递归/函数指针返回/数组指针返回。 */

void no_args(void) {
}

void empty_parens() {
}

int variadic(int first, ...) {
    return first;
}

static int static_helper(int v) {
    return v;
}

inline int inline_helper(int v) {
    return v;
}

int fact(int n) {
    if (n <= 1) {
        return 1;
    }
    return n * fact(n - 1);
}

/* 函数指针变量：不是函数声明 */
int (*gfn)(int);

/* 返回函数指针的函数：是函数定义 */
int (*rfp(int n))(int) {
    return gfn;
}

/* 返回数组指针的函数：是函数定义 */
int (*rap(void))[5] {
    return 0;
}

/* 返回函数指针的函数原型 */
void (*signal_like(int sig, void (*handler)(int)))(int);
