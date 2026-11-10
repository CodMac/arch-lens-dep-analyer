/* 基准元素：C++ 16 种 ElementKind 各至少一例的最小形态。 */
#ifndef CPP_BASE_H
#define CPP_BASE_H

#include <cstddef>
#include "cpp_base_dep.h"

#define BASE_MACRO 42
#define BASE_FUNC_MACRO(x) ((x) + 1)

namespace base_ns {

struct BaseStruct {
    int field;
};

class BaseClass {
public:
    int member;
    void method();
    BaseClass();
};

enum BaseEnum {
    BE_A,
    BE_B
};

enum class ScopedEnum : unsigned char {
    SE_A,
    SE_B
};

typedef unsigned long base_ulong_t;
using BaseAlias = int;

int base_func(int a);

extern int base_global;

} // namespace base_ns

#endif // CPP_BASE_H
