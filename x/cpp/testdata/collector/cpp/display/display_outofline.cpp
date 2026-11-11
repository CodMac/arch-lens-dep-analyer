/* 展示层字段：跨文件类外定义（有 namespace 块 / 无 namespace 块）。 */
#include "display_outofline.h"

namespace ool {

int Service::member(int x) { return x; }

int Service::staticMember(int x) { return x + 1; }

Service::Service() {}

Service::~Service() {}

Service &Service::operator=(const Service &o) { return *this; }

bool Service::operator==(const Service &o) const { return true; }

int freeFn(int a) { return a; }

namespace nested {

int Inner::innerMethod(int x) { return x * 2; }

} // namespace nested

} // namespace ool

/* 无 namespace 块：仅限定定义。展示层应从 QN 结构推导 namespace 与宿主类型。 */
int GlobalCls::method(int x) { return x; }

int globalFreeFn(int a) { return a; }

/* 声明侧在 ool_no_block 命名空间，但本文件不写 namespace 块。 */
int ool_no_block::Remote::method(int x) { return x; }

ool_no_block::Remote::Remote() {}

ool_no_block::Remote::~Remote() {}
