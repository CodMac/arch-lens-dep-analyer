/* 作用域：嵌套 block、if/for/while/switch/try 体、函数内局部 class/enum、块内重名。 */

void scope_user(int n) {
    int fn_level = 0;

    /* 裸 block */
    {
        int in_bare = 1;
    }

    /* if 体 */
    if (n > 0) {
        int in_if = 1;
    } else {
        int in_else = 2;
    }

    /* for 体 */
    for (int i = 0; i < n; ++i) {
        int in_for = i;
    }

    /* range-for 体 */
    int arr[3] = {1, 2, 3};
    for (int v : arr) {
        int in_range = v;
    }

    /* while 体 */
    while (n > 0) {
        int in_while = 1;
        n--;
    }

    /* do-while 体 */
    do {
        int in_do = 1;
    } while (false);

    /* switch 体 */
    switch (n) {
    case 0: {
        int in_case = 0;
        break;
    }
    default: {
        int in_default = 1;
        break;
    }
    }

    /* try / catch 体 */
    try {
        int in_try = 1;
    } catch (...) {
        int in_catch = 2;
    }

    /* 嵌套 block */
    {
        int outer_block = 1;
        {
            int inner_block = 2;
        }
    }

    /* 块内重名：不同作用域各自独立 */
    {
        int dup_name = 1;
    }
    {
        int dup_name = 2;
    }

    /* 函数内局部 class / enum */
    struct LocalStruct {
        int lv;
    };
    class LocalClass {
    public:
        int lc;
    };
    enum LocalEnum {
        LE_A
    };
}
