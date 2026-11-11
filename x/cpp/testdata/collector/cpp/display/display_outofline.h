/* 展示层字段：跨文件类外定义的声明侧。 */
#ifndef DISPLAY_OUTOFLINE_H
#define DISPLAY_OUTOFLINE_H

namespace ool {

struct Service {
    Service();
    ~Service();
    int member(int x);
    static int staticMember(int x);
    Service &operator=(const Service &o);
    bool operator==(const Service &o) const;
};

int freeFn(int a);

namespace nested {
struct Inner {
    int innerMethod(int x);
};
} // namespace nested

} // namespace ool

/* 无 namespace 块的声明侧（供无 namespace 块的源文件配对） */
struct GlobalCls {
    int method(int x);
};

int globalFreeFn(int a);

/* 有 namespace 声明，但对应源文件不写 namespace 块（仅限定定义） */
namespace ool_no_block {

struct Remote {
    int method(int x);
    Remote();
    ~Remote();
};

} // namespace ool_no_block

#endif // DISPLAY_OUTOFLINE_H
