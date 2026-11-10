/* 继承：单/多继承、virtual 继承、public/private/protected 继承、override/final。 */

struct BaseA {
    int a;
    virtual void fa();
    virtual ~BaseA() {}
};

struct BaseB {
    int b;
    virtual void fb();
};

struct BaseC {
    int c;
};

/* 单继承 */
struct Single : public BaseA {
    int s;
};

/* 多继承 */
struct Multi : public BaseA, public BaseB {
    int m;
};

/* virtual 继承（菱形） */
struct VLeft : virtual public BaseA {
    int l;
};

struct VRight : virtual public BaseA {
    int r;
};

struct Diamond : public VLeft, public VRight {
    int d;
};

/* private / protected 继承 */
struct PrivDerived : private BaseB {
    int p;
};

struct ProtDerived : protected BaseC {
    int q;
};

/* override / final */
struct Overrider : public BaseA {
    void fa() override;
    void extra() final;
};

struct FinalDerived final : public BaseB {
    void fb() override;
};

/* 继承 + 类内字段/方法 */
struct Rich : public BaseA, protected BaseC {
    int rich_field;
    void rich_method();
};
