/* 变量与作用域：全局/static/const/extern、局部、块作用域、多声明器、数组、初始化。 */

int g_counter;

static int g_static;

const int g_const = 7;

int g_arr[8];

int g_multi1, g_multi2;

int g_init = 3;

void scope_fn(int param) {
    int local = 1;
    static int local_static = 2;
    const int local_const = 3;
    int local_multi_a, local_multi_b;
    int local_arr[4];

    {
        int inner = 4;
        (void)inner;
    }

    for (int i = 0; i < 3; i++) {
        int loop_body = i;
        (void)loop_body;
    }

    (void)local;
    (void)local_static;
    (void)local_const;
    (void)local_multi_a;
    (void)local_multi_b;
    (void)local_arr;
    (void)param;
}

/* 混合声明器：函数 + 变量共存 */
int mixf(int a), mixv;
