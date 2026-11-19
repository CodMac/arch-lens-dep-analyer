/* 结构关系样例：源文件侧定义 + 系统头 + 嵌套路径 include。 */
#include "structural.h"
#include "nested/deep.h"
#include <vector>

namespace st {

int compute(Dep d, int n) {
    return d.v + n;
}

Dep make_dep() {
    Dep d;
    d.v = 1;
    return d;
}

void no_return(Dep d) {
    (void)d;
}

void take_void(void) {
}

Dep g_dep;
int g_count = 0;

/* 变量类型 + 模板实参 */
void use_holder() {
    Holder<Dep> h;
    h.item = g_dep;
    (void)h;
}

/* 嵌套命名空间类型 */
void use_deep() {
    nested::Deep d;
    d.d = 1;
    (void)d;
}

} // namespace st
