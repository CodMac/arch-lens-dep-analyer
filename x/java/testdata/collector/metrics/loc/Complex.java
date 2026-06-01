package /* 内部块 */ com.example;

// (预期 LOC: 6)
public class Main {
    public void test() {
        /*
        跨行块注释
        // 里面还有单行注释符
        */
        String s = "/* 字符串里的假注释 */";
    }
}