/* 模板：类模板/函数模板、可变参数、默认参数、全特化/偏特化、别名模板、concept。 */

/* 类模板 */
template <typename T>
struct Box {
    T value;
    void put(T v);
};

/* 函数模板 */
template <typename T>
T identity(T v) {
    return v;
}

/* 多参数 + 默认模板参数 */
template <typename K, typename V = int, int N = 8>
struct Map {
    K key;
    V val;
};

/* 可变参数模板 */
template <typename... Ts>
struct Tuple {
    int size;
};

template <typename... Ts>
int count_args(Ts... args) {
    return sizeof...(Ts);
}

/* 全特化：QN 剥离实参 → Box */
template <>
struct Box<int> {
    int int_value;
};

/* 偏特化：QN 剥离实参 → Box */
template <typename T>
struct Box<T*> {
    T* ptr_value;
};

/* 函数模板显式特化：QN 剥离实参 → identity(int) */
template <>
int identity<int>(int v) {
    return v;
}

/* 别名模板 */
template <typename T>
using BoxAlias = Box<T>;

/* concept / requires（C++20） */
template <typename T>
concept Small = sizeof(T) <= 4;

template <Small T>
T constrained_func(T v) {
    return v;
}

/* 类模板成员类外定义 */
template <typename T>
void Box<T>::put(T v) {
    value = v;
}
