/* 构造/析构：默认、= default、= delete、带参、explicit、拷贝、移动、委托、成员初始化列表。 */

class Widget {
public:
    Widget();
    Widget(int v);
    explicit Widget(const char* name);
    Widget(const Widget& other);
    Widget(Widget&& other) noexcept;
    Widget(int v, int w);
    ~Widget();

    Widget& operator=(const Widget& other);
    Widget& operator=(Widget&& other) noexcept;

private:
    int value_;
    const char* name_;
};

class Defaulted {
public:
    Defaulted() = default;
    ~Defaulted() = default;
    Defaulted(const Defaulted&) = default;
    Defaulted& operator=(const Defaulted&) = default;
};

class Deleted {
public:
    Deleted() = delete;
    Deleted(const Deleted&) = delete;
    Deleted& operator=(const Deleted&) = delete;
};

class Delegating {
public:
    Delegating() : Delegating(0) {}
    Delegating(int v) : value_(v) {}

private:
    int value_;
};

class Inline {
public:
    Inline() : a_(1), b_(2) {}
    ~Inline() {}

private:
    int a_;
    int b_;
};

/* 类外定义。 */
Widget::Widget() : value_(0), name_(nullptr) {}
Widget::Widget(int v) : value_(v), name_(nullptr) {}
Widget::~Widget() {}
