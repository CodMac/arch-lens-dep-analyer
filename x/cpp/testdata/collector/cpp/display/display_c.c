/* 展示层字段：纯 C 文件（无 namespace 概念，NamespaceQN 恒空）。 */
#include <stdio.h>

struct CStruct {
    int a;
};

typedef struct {
    int x;
    int y;
} CTypedef;

enum CEnum {
    C_A,
    C_B
};

union CUnion {
    int ui;
    float uf;
};

int c_global;
static int c_static;

void c_func(void) {
    int local = 0;
    struct CLocal {
        int l;
    };
    CLocal cl;
    (void)local;
    (void)cl;
}
