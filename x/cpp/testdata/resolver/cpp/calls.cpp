namespace util {
int helper();
}

enum Color { RED, GREEN };

int free_func(int x);

class Widget {
public:
    int count;

    void draw();
    void resize(int w);
    static Widget* create();
};

void caller() {
    int local = free_func(1);
    Widget w;
    w.draw();
    w.resize(local);
    Widget* ptr = Widget::create();
    ptr->draw();
    w.count = local;
    int read = w.count;
    Color c = Color::RED;
    util::helper();
    (void)read;
    (void)c;
}

void maker() {
    Widget* created = new Widget();
    (void)created;
}

void caster(void* raw) {
    Widget* w = (Widget*)raw;
    Widget* s = static_cast<Widget*>(raw);
    (void)w;
    (void)s;
}

int capture_user(int seed) {
    int base = seed + 1;
    auto add_base = [base](int v) { return v + base; };
    return add_base(2);
}

struct Pair2 {
    int a;
    int b;
    Pair2(int x, int y) : a(x), b(y) {}
};
