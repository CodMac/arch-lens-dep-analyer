/* 条件编译：if/ifdef/ifndef/elif/else、嵌套条件取最内层、#pragma。 */

/* #ifdef */
#ifdef FEATURE_A
int cond_a;
#endif

/* #ifndef */
#ifndef FEATURE_B
int cond_b;
#endif

/* #if */
#if defined(FEATURE_C) && FEATURE_C > 0
int cond_c;
#endif

/* #elif 链 */
#if FEATURE_D == 1
int cond_d1;
#elif FEATURE_D == 2
int cond_d2;
#else
int cond_d3;
#endif

/* #else */
#ifdef FEATURE_E
int cond_e1;
#else
int cond_e2;
#endif

/* 嵌套条件：取最内层标签 */
#ifdef OUTER_F
#ifdef INNER_F
int cond_nested;
#endif
int cond_outer_only;
#endif

/* 条件分支内的函数与类型 */
#ifdef FEATURE_G
struct CondStruct {
    int cv;
};
void cond_func();
#endif

/* #pragma 不产元素 */
#pragma once
#pragma pack(push, 1)
#pragma pack(pop)
