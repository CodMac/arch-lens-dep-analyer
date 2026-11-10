// 链式调用解析样例：接收者类型沿调用返回类型 / 字段声明类型传播。
// 覆盖 f()->g()、a.b.c、this->f()->g()、自由函数返回值作接收者。

struct L3 {
    int v;
    int deep() { return v; }
};

struct L2 {
    L3 *l3;
    L3 *getL3() { return l3; }
};

struct L1 {
    L2 *l2;
    L2 *getL2() { return l2; }
};

struct Holder {
    L1 *l1;
    int tag;
};

L1 *makeL1(Holder *h) { return h->l1; }

// 同名方法在不同类型上：用于验证接收者类型传播而非全局短名兜底。
struct Alpha {
    int pick() { return 1; }
};

struct Beta {
    int pick() { return 2; }
};

struct Switcher {
    Alpha *a;
    Beta *b;
    Alpha *getAlpha() { return a; }
    Beta *getBeta() { return b; }
};

void chain_use(Holder *h, Switcher *s) {
    int a = h->l1->l2->l3->deep();
    int b = h->l1->getL2()->getL3()->deep();
    int c = makeL1(h)->getL2()->getL3()->deep();
    int d = s->getAlpha()->pick();
    int e = s->getBeta()->pick();
    int f = s->a->pick();
    int g = s->b->pick();
    int t = h->tag;
    (void)a; (void)b; (void)c; (void)d; (void)e; (void)f; (void)g; (void)t;
}

class Owner {
public:
    Alpha *a;
    Beta *b;
    Alpha *self() { return a; }
    int run() {
        int x = this->self()->pick();
        int y = this->b->pick();
        return x + y;
    }
};
