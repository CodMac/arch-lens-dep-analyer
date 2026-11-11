/* 展示层字段：namespace 全形态（嵌套 / C++17 / inline / 匿名 / 重开 / alias）。 */

namespace ns_outer {
namespace ns_middle {
namespace ns_inner {
struct DeepType {
    int d;
};
} // namespace ns_inner
} // namespace ns_middle
} // namespace ns_outer

/* C++17 嵌套写法 */
namespace ns_modern::ns_sub {
struct ModernType {
    int m;
};
} // namespace ns_modern::ns_sub

/* inline namespace */
namespace ns_inline {
inline namespace v1 {
struct InlineType {
    int i;
};
} // namespace v1
} // namespace ns_inline

/* 匿名 namespace */
namespace {
int anon_var;
struct AnonType {
    int af;
};
} // namespace

/* 重开 namespace：QN 层为 re$2，展示层 NamespaceQN 应归并为 re */
namespace ns_reopen {
struct First {
    int a;
};
} // namespace ns_reopen

namespace ns_reopen {
struct Second {
    int b;
};
} // namespace ns_reopen

/* namespace alias 不产元素 */
namespace ns_alias = ns_outer::ns_middle;
