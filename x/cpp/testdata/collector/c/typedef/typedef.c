/* typedef：基础别名、指针别名、数组别名、函数指针别名、多声明器、链式。 */

typedef unsigned long ulong_t;

typedef int *int_ptr_t;

typedef int int_arr_t[10];

typedef int (*callback_t)(int, int);

/* 多声明器：每个都产 TYPE */
typedef int TI1, TI2;

typedef int TI3, *PTI3;

/* 链式别名 */
typedef ulong_t my_ulong_t;

/* 匿名 struct typedef */
typedef struct {
    int x;
    int y;
} Point;

/* 匿名 struct 多声明器 typedef */
typedef struct {
    int a;
} TA, *PTA;

/* 具名 struct typedef */
typedef struct Named {
    int n;
} Named_t;
