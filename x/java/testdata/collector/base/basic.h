#ifndef BASIC_H
#define BASIC_H

#include <stdio.h>
#include "util.h"

#define MAX_SIZE 100
#define SQUARE(x) ((x) * (x))

typedef struct {
    int x;
    int y;
} Point;

typedef unsigned long ulong_t;

struct Node {
    int value;
    struct Node* next;
};

enum Color {
    RED,
    GREEN,
    BLUE
};

int add(int a, int b);

extern int global_counter;

#endif
