/* ImplLink 样例：1 头 ↔ 多源、static 不配对、只声明不定义、类外定义。 */
#ifndef IMPL_LINK_IMPL_H
#define IMPL_LINK_IMPL_H

namespace impl {

/* 1 头 ↔ 多源：声明在此，定义分散在 impl_a.cpp / impl_b.cpp */
int split_a(int x);
int split_b(int x);

/* 正常配对 */
int paired(int x);

/* static 函数（源文件内定义，无声明）——不参与配对 */
/* extern 声明但无定义——不建 ImplLink */
int declared_only(int x);

class Service {
public:
    int member(int x);
    int inline_member(int x) { return x; }
};

/* 类外定义配对 */
int Service::member(int x);

} // namespace impl

#endif // IMPL_LINK_IMPL_H
