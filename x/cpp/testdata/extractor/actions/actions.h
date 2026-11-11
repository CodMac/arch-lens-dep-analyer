/* 动作关系样例：Call / Create / Cast / Assign / Use / Throw。 */
#ifndef ACTIONS_H
#define ACTIONS_H

namespace act {

struct Target {
    int field;
    int get() const;
    void set(int v);
};

struct Base {
    virtual ~Base() {}
    virtual int run() { return 1; }
};

struct Derived : public Base {
    int run() override { return 2; }
};

int free_fn(int a);
int free_fn(int a, int b);
int ns_free();

namespace inner {
int ns_free();
}

class Widget {
public:
    int count;
    Widget();
    Widget(int v);
    ~Widget();
    void draw();
    static Widget *create();

    Widget &operator=(const Widget &o);
    int operator()(int x);
};

/* 异常类型 */
struct MyError {
    int code;
};

void thrower(int mode);

} // namespace act

#endif // ACTIONS_H
