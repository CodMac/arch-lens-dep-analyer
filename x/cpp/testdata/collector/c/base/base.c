/* 基准元素：文件/函数/变量/结构体/枚举/宏/typedef 的最小形态。 */
#include <stdio.h>
#include "base.h"

#define ANSWER 42

int counter;

struct Point {
    int x;
    int y;
};

enum Color {
    RED,
    GREEN
};

typedef unsigned long ulong_t;

int add(int a, int b) {
    int sum = a + b;
    return sum;
}
