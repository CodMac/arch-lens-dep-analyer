/* 命名空间族：嵌套、C++17 嵌套写法、inline、匿名、namespace alias、using。 */

namespace outer {
namespace middle {
namespace inner {
struct Deep {
    int d;
};
} // namespace inner
} // namespace middle
} // namespace outer

/* C++17 嵌套命名空间定义：a::b 应拆成 a 与 a::b 两级。 */
namespace modern::sub {
struct Modern {
    int m;
};
} // namespace modern::sub

/* inline namespace：元素仍挂在 inline 命名空间下。 */
namespace inl {
inline namespace v1 {
int inline_var;
} // namespace v1
} // namespace inl

/* 匿名 namespace：记名 $anon。 */
namespace {
int anon_var;
struct AnonStruct {
    int a;
};
} // namespace

/* namespace alias 不产元素（不是定义）。 */
namespace alias_ns = outer::middle;

/* using 声明/指令不产元素。 */
using namespace outer;
using outer::middle::inner::Deep;

int use_all() {
    Deep d;
    alias_ns::inner::Deep e;
    return d.d + e.d + anon_var + inl::inline_var;
}
