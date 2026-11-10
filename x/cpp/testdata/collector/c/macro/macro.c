/* 宏与预处理：对象宏、函数式宏、空宏、多行宏、条件包含。 */

#define MAX_SIZE 100

#define SQUARE(x) ((x) * (x))

#define EMPTY_MACRO

#define MULTI_LINE(a, b) \
    ((a) + (b))

#define STR(x) #x

#define CAT(a, b) a##b

#include <stdio.h>
#include "macro.h"

#ifdef FEATURE_X
#define FEATURE_VALUE 1
#else
#define FEATURE_VALUE 0
#endif
