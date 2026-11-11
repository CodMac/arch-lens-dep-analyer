/* Resolver 强转样例。 */
#include "cast.h"

namespace cst {

void cast_all(void *raw, CastPoly *poly) {
    CastA *c_style = (CastA *)raw;
    CastA *s_cast = static_cast<CastA *>(raw);
    CastPolyD *d_cast = dynamic_cast<CastPolyD *>(poly);
    const CastA *k_cast = const_cast<const CastA *>(c_style);
    CastB *r_cast = reinterpret_cast<CastB *>(raw);

    /* (void)x 抑制形态：不产 CAST */
    (void)c_style;
    (void)s_cast;
    (void)d_cast;
    (void)k_cast;
    (void)r_cast;
}

} // namespace cst
