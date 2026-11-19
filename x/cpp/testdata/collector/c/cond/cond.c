/* 条件编译：ifdef/ifndef/if/elif/else 嵌套，全分支保留并标注 condition。 */

#ifdef FOO
int cond_a = 1;
#elif defined(BAR)
int cond_b = 2;
#else
int cond_c = 3;
#endif

#ifndef BAZ
void cond_d(void) {}
#endif

#ifndef FOO
int cond_ifndef;
#endif

#if defined(BAR)
int cond_if;
#elif defined(BAZ)
int cond_elif;
#else
int cond_else;
#endif

#ifdef OUTER
#ifdef INNER
int cond_nested;
#endif
#endif

#ifdef FOO
int cond_fn(void) {
    return 1;
}
#endif
