#include "basic.h"

int global_counter = 0;

static int internal_helper(int v) {
    return v + 1;
}

int add(int a, int b) {
    int local = a + b;
    return local;
}

void process(void) {
    Point p;
    p.x = add(1, 2);
    struct Node n;
    n.value = internal_helper(p.x);
    enum Color c = RED;
    (void)c;
}
