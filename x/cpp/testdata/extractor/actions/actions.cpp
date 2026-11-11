/* 动作关系样例：各动作形态。 */
#include "actions.h"

namespace act {

int free_fn(int a) { return a; }
int free_fn(int a, int b) { return a + b; }

int ns_free() { return 0; }

namespace inner {
int ns_free() { return 1; }
}

int Target::get() const { return field; }
void Target::set(int v) { field = v; }

Widget::Widget() : count(0) {}
Widget::Widget(int v) : count(v) {}
Widget::~Widget() {}

void Widget::draw() {}

Widget *Widget::create() { return new Widget(); }

Widget &Widget::operator=(const Widget &o) {
    count = o.count;
    return *this;
}

int Widget::operator()(int x) { return x + count; }

/* --- Call 各接收者形态 --- */
int caller(int n) {
    int a = free_fn(n);                    // 自由函数（重载按实参个数）
    int b = free_fn(n, 1);
    int c = ns_free();                     // 当前命名空间
    int d = inner::ns_free();              // 限定命名空间
    Widget w;
    w.draw();                              // 成员
    Widget *p = Widget::create();          // 静态成员
    p->draw();                             // 指针成员
    Target t;
    t.set(n);
    int e = t.get();                       // const 成员
    int f = w(n);                          // operator()
    return a + b + c + d + e + f;
}

/* --- Create + 构造器 CALL --- */
Widget *maker() {
    Widget *a = new Widget();
    Widget *b = new Widget(3);
    return a;
}

/* --- Cast 各形态 --- */
void caster(void *raw, Base *bp) {
    Target *c1 = (Target *)raw;            // C 风格
    Target *c2 = static_cast<Target *>(raw);
    Base *c3 = dynamic_cast<Base *>(raw);
    const Target *c4 = const_cast<const Target *>(c2);
    Target *c5 = reinterpret_cast<Target *>(raw);
    (void)bp;
    (void)c1; (void)c2; (void)c3; (void)c4; (void)c5;
}

/* --- Assign 各形态 --- */
void assigner(Target *t, int n) {
    int local = n;                         // 初始化
    local = n + 1;                         // 赋值
    t->field = local;                      // 字段赋值
    local += 2;                            // 复合赋值
    Target copy;
    copy = *t;                             // 拷贝赋值
    (void)copy;
    (void)local;
}

/* --- Use 各形态 --- */
int user(Target *t, int n) {
    int local = n;
    int a = t->field;                      // 字段读取
    int b = local;                         // 变量读取
    int c = t->get();                      // 方法调用（非 USE）
    return a + b + c;
}

/* --- Throw 各形态 --- */
void thrower(int mode) {
    if (mode == 0) {
        throw new MyError();               // throw new T
    }
    if (mode == 1) {
        throw MyError();                   // throw T(...)
    }
    MyError e;
    e.code = mode;
    throw e;                               // throw var
}

} // namespace act
