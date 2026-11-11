/* Resolver 强转样例：C 风格 + 四种显式转换关键字 + (void) 抑制。 */
#ifndef RESOLVER_CAST_H
#define RESOLVER_CAST_H

namespace cst {

struct CastA {
    int a;
};

struct CastB {
    int b;
};

struct CastPoly {
    virtual ~Poly() {}
};

struct CastPolyD : public CastPoly {
    int d;
};

} // namespace cst

#endif // RESOLVER_CAST_H
