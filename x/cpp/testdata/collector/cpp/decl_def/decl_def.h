/* 声明/定义分离：头文件原型、前向声明、extern 变量。 */
#ifndef CPP_DECL_DEF_H
#define CPP_DECL_DEF_H

struct FwdStruct;
class FwdClass;
enum FwdEnum : int;

struct DefinedInHeader {
    int hv;
};

class DeclaredClass {
public:
    void declared_method();
    int declared_field;
};

int header_func(int a);

extern int extern_var;

#endif // CPP_DECL_DEF_H
