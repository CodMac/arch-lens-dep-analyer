/* 枚举：普通 enum、enum class、指定底层类型、前向声明、匿名枚举、显式值。 */

/* 普通 enum */
enum Color {
    RED,
    GREEN,
    BLUE
};

/* enum class */
enum class Scoped {
    SA,
    SB,
    SC
};

/* 指定底层类型 */
enum class Small : unsigned char {
    MIN = 0,
    MAX = 255
};

enum Tiny : short {
    T_NEG = -1,
    T_POS = 1
};

/* 前向声明 + 定义 */
enum FwdEnum : int;
enum FwdEnum : int {
    FE_A,
    FE_B
};

/* 匿名 enum：常量挂文件级，不产 ENUM 元素 */
enum {
    ANON_X,
    ANON_Y
};

/* 显式值 */
enum Flags {
    F_NONE = 0,
    F_READ = 1 << 0,
    F_WRITE = 1 << 1,
    F_ALL = F_READ | F_WRITE
};

/* 枚举类型变量 */
Color current_color;
Scoped current_scoped;
