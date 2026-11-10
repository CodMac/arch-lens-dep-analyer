/* 声明与定义分离：同文件声明+定义、头文件原型、前向声明、extern。 */
#include "decl_def.h"

/* 同文件先声明后定义：共享同一 QN，不产生 $2 */
int proto(int a);
int proto(int a) {
    return a;
}

/* 结构体前向声明 + 后续定义 */
struct FwdDef;
struct FwdDef {
    int v;
};

/* 枚举前向声明（C23 合法；tree-sitter 接受）+ 定义 */
enum FwdEnum;
enum FwdEnum {
    FE_A
};

/* extern 变量声明 */
extern int shared_var;
int shared_var = 0;

/* 只声明不定义 */
int never_defined(int x);
