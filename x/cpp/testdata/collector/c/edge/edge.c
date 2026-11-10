/* 边界与负例：重名、空文件元素、类型引用不产元素、无名参数、嵌套作用域重名。 */

struct Tag {
    int v;
};

/* 局部变量使用 struct Tag：只产变量，不产新的 STRUCT 元素 */
void use_type_ref(void) {
    struct Tag t;
    enum { LOCAL_A } local_e;
    (void)t;
    (void)local_e;
}

/* 同作用域重名局部变量（不同块） */
void dup_blocks(void) {
    {
        int dup = 1;
        (void)dup;
    }
    {
        int dup = 2;
        (void)dup;
    }
}

/* 同名函数不同签名 */
int overload_like(int a) {
    return a;
}

int overload_like2(int a, int b) {
    return a + b;
}

/* 指针参数无名 */
void no_name_ptr(char *) {
}

/* 嵌套 struct 定义（具名） */
struct NestedOuter {
    struct NestedInner {
        int deep;
    } inner;
};
