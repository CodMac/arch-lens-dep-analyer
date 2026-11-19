/* Resolver throw 样例：throw new T / throw T(...) / throw var / rethrow。 */
#ifndef RESOLVER_THROW_H
#define RESOLVER_THROW_H

namespace thr {

struct ErrA {
    int code;
};

struct ErrB {
    int code;
};

void thrower(int mode);

} // namespace thr

#endif // RESOLVER_THROW_H
