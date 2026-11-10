/* enum：具名、显式值、匿名、前向声明、枚举常量引用。 */

enum Color {
    RED,
    GREEN,
    BLUE
};

enum Status {
    OK = 0,
    WARN = 10,
    FAIL = 20
};

/* 匿名 enum：常量挂文件级，不产 ENUM 元素 */
enum {
    ANON_A,
    ANON_B
};

/* 前向声明 */
enum Forward;

enum Forward {
    FWD_A
};

/* 枚举常量作为变量初值 */
enum Color current = RED;
