/* ImplLink 样例源 A：split_a / paired / static 函数 / 类外定义。 */
#include "impl.h"

namespace impl {

int split_a(int x) { return x + 1; }

int paired(int x) { return x * 2; }

static int file_local(int x) { return x - 1; }

int Service::member(int x) { return x + 100; }

int use_local(int x) { return file_local(x); }

} // namespace impl
