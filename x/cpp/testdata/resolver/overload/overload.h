/* Resolver 重载挑选样例：按实参个数选择；定义优先于声明。 */
#ifndef RESOLVER_OVERLOAD_H
#define RESOLVER_OVERLOAD_H

namespace ov {

/* 同文件声明 + 定义：定义优先 */
int pick(int a);
int pick(int a, int b);
int pick();

struct Svc {
    int call(int a);
    int call(int a, int b);
    int call();
};

} // namespace ov

#endif // RESOLVER_OVERLOAD_H
