/* Resolver throw 样例。 */
#include "throw.h"

namespace thr {

void thrower(int mode) {
    if (mode == 0) {
        throw new ErrA();       // throw new T -> 类型 ErrA
    }
    if (mode == 1) {
        throw ErrB();           // throw T(...) -> 类型 ErrB
    }
    ErrA e;
    e.code = mode;
    throw e;                    // throw var -> 变量声明类型 ErrA
}

} // namespace thr
