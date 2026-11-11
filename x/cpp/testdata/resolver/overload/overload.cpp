/* Resolver 重载挑选样例。 */
#include "overload.h"

namespace ov {

int pick(int a) { return a; }
int pick(int a, int b) { return a + b; }
int pick() { return 0; }

int Svc::call(int a) { return a; }
int Svc::call(int a, int b) { return a + b; }
int Svc::call() { return 0; }

void overload_user(Svc *s) {
    int a = pick(1);        // pick(int)
    int b = pick(1, 2);     // pick(int,int)
    int c = pick();         // pick()
    int d = s->call(1);     // Svc::call(int)
    int e = s->call(1, 2);  // Svc::call(int,int)
    int f = s->call();      // Svc::call()
    (void)a; (void)b; (void)c; (void)d; (void)e; (void)f;
}

} // namespace ov
