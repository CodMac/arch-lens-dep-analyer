#include "basic.h"

namespace app {

namespace core {

class Base {
public:
    virtual int run(int x) = 0;
    static const int K = 1;

protected:
    int m_val;

private:
    void helper();
};

} // namespace core

struct Point3 {
    int x;
    int y;
    int z;
};

enum class Scoped {
    A,
    B
};

template <typename T>
class Box : public core::Base {
public:
    T value;

    int run(int x) override {
        return x + value;
    }

    int twice(int x);
};

int core::Base::helper_impl = 0;

void core::Base::helper() {
    m_val = 1;
}

} // namespace app

int app::Box<int>::twice(int x) {
    return x * 2;
}

void caller() {
    app::Point3 p;
    p.x = 1;
    app::Scoped s = app::Scoped::A;
    (void)s;
}
