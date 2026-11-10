/* 类族：访问级别、嵌套类型、friend、static/const/mutable、final、抽象类。 */

class Abstract {
public:
    virtual void pure() = 0;
    virtual ~Abstract() {}
};

class FinalClass final {
public:
    int fv;
};

class Outer {
public:
    struct NestedStruct {
        int ns;
    };

    class NestedClass {
    public:
        int nc;
    };

    enum NestedEnum {
        NE_A
    };

    enum class NestedScoped {
        NS_A
    };

    typedef int nested_alias_t;

    static int static_field;
    static const int static_const_field = 1;
    const int const_field = 2;
    mutable int mutable_field;

    void pub_method();

    friend void friend_func(Outer&);

protected:
    int prot_field;
    void prot_method();

private:
    int priv_field;
    void priv_method();

public:
    class Inner {
    public:
        int deep;
    };
};

void friend_func(Outer& o) {
    o.priv_field = 0;
}

/* 类内 static 成员定义。 */
int Outer::static_field = 0;
