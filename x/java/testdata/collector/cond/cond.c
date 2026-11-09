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
