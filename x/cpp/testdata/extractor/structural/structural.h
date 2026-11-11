/* 结构关系样例：Import / Contain / Extend / Parameter / Return / TypeArg。 */
#ifndef STRUCTURAL_H
#define STRUCTURAL_H

#include <string>
#include "structural_dep.h"

namespace st {

/* Extend：单继承 */
struct Base {
    virtual ~Base() {}
};

/* Extend：多继承（跨文件类型 + 同文件类型） */
struct Mid : public Base {
    int m;
};

struct Leaf : public Mid, public Dep {
    int l;
};

/* TypeArg：字段自身类型 + 模板实参 */
template <typename T>
struct Holder {
    T item;
    Dep *dep;
    Holder<Dep> *nested;
};

enum Kind {
    K_A,
    K_B
};

/* Parameter / Return：源码类型 */
int compute(Dep d, int n);
Dep make_dep();

/* Parameter / Return：void 返回不产 Return */
void no_return(Dep d);

/* 无参 */
void take_void(void);

/* typedef / using 别名 */
typedef Dep DepAlias;
using DepUsing = Dep;

/* 全局变量类型 */
extern Dep g_dep;
extern int g_count;

} // namespace st

#endif // STRUCTURAL_H
