/* 运算符重载：算术/比较/赋值/下标/调用/自增自减/转换/内存分配，成员与非成员。 */

class Num {
public:
    Num() : v_(0) {}
    explicit Num(int v) : v_(v) {}

    /* 算术 */
    Num operator+(const Num& o) const;
    Num operator-(const Num& o) const;
    Num operator*(int k) const;
    Num& operator+=(const Num& o);

    /* 比较 */
    bool operator==(const Num& o) const;
    bool operator!=(const Num& o) const;
    bool operator<(const Num& o) const;
    bool operator<=(const Num& o) const;

    /* 赋值 */
    Num& operator=(const Num& o);
    Num& operator=(Num&& o) noexcept;

    /* 下标 / 调用 */
    int& operator[](int i);
    const int& operator[](int i) const;
    void operator()();
    int operator()(int a, int b);

    /* 自增自减 */
    Num& operator++();
    Num operator++(int);
    Num& operator--();
    Num operator--(int);

    /* 转换运算符 */
    explicit operator bool() const;
    operator int() const;

    /* 内存分配 */
    static void* operator new(unsigned long size);
    static void operator delete(void* p);

    /* 逻辑 / 位运算 */
    bool operator!() const;
    Num operator&(const Num& o) const;
    Num operator|(const Num& o) const;
    Num operator~() const;

private:
    int v_;
};

/* 非成员运算符 */
Num operator+(int k, const Num& n);
bool operator>(const Num& a, const Num& b);
