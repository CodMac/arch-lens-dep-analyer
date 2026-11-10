/* 宏与预处理：对象宏、函数式宏、可变参数宏、多行宏、空宏、#undef、条件分支内同名宏。 */

/* 对象宏 */
#define OBJ_MACRO 100

/* 函数式宏 */
#define FUNC_MACRO(x) ((x) * 2)

/* 多参数函数式宏 */
#define ADD_MACRO(a, b) ((a) + (b))

/* 可变参数宏 */
#define VAR_MACRO(fmt, ...) printf(fmt, __VA_ARGS__)

/* 多行宏 */
#define MULTI_LINE(x) \
    do {             \
        (x)++;       \
    } while (0)

/* 空宏 */
#define EMPTY_MACRO

/* 字符串化 / 连接 */
#define STR_MACRO(x) #x
#define CAT_MACRO(a, b) a##b

/* 条件分支内同名宏：全分支保留，各自标注 condition */
#ifdef FEATURE_Y
#define FEATURE_VAL 1
#else
#define FEATURE_VAL 0
#endif

/* #undef 不产元素 */
#undef OBJ_MACRO

/* 宏使用 */
int use_macros() {
    int v = OBJ_MACRO;
    v = FUNC_MACRO(v);
    v = ADD_MACRO(v, 1);
    return v;
}
