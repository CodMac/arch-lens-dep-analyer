/* 参数形态：具名/无名、指针/多级指针、数组、函数指针、const、struct/enum 关键字。 */

struct S {
    int v;
};

enum E {
    E_A
};

typedef int (*cb_t)(int);

void named(int a, char *p) {
    (void)a;
    (void)p;
}

void unnamed(int, char *, struct S) {
}

void multilevel(int **pp, char ***ppp) {
    (void)pp;
    (void)ppp;
}

void arrays(int arr[], char buf[16]) {
    (void)arr;
    (void)buf;
}

void func_ptr_param(cb_t cb, int (*direct)(int)) {
    (void)cb;
    (void)direct;
}

void qualified(const int *cp, const struct S *sp, enum E e) {
    (void)cp;
    (void)sp;
    (void)e;
}

void mixed(int first, ...) {
    (void)first;
}
