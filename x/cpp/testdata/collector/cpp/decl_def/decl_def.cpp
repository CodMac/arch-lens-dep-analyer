/* 声明/定义分离：同文件声明+定义、类外定义、前向声明转定义。 */
#include "decl_def.h"

/* 同文件先声明后定义：共享同一 QN，不产生 $2。 */
int local_proto(int a);
int local_proto(int a) {
    return a;
}

/* 前向声明 + 定义：定义覆盖声明。 */
struct FwdStruct {
    int fv;
};

class FwdClass {
public:
    int cv;
};

enum FwdEnum : int {
    FE_A,
    FE_B
};

/* 头文件原型对应的定义。 */
int header_func(int a) {
    return a;
}

/* 类内声明 + 类外定义。 */
void DeclaredClass::declared_method() {
    declared_field = 1;
}

/* extern 声明 + 定义。 */
int extern_var = 0;

/* 只声明不定义：仅产 FunctionDecl。 */
int never_defined(int a);
