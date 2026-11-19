/* 匿名 struct 承载 typedef 的字段解析样例：
   typedef struct {...} Point; 的变量 p 其 type_with_qn 必须是 Point（而非匿名结构体文本），
   否则 p.x 会退化为全局短名扫描，命中同名参数 x（并行收集顺序不定 → 结果不稳定）。 */

typedef struct {
    int x;
    int y;
} Point;

void process(void) {
    Point p;
    p.x = 1;
    int read = p.x;
    (void)read;
}
