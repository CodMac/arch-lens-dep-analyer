package com.example.metrics;

public class ComplexityEdge {
    // 空方法 base=1
    public void base() {}

    // do_statement +1, while +1 => 3
    public void doLoop() {
        int i = 0;
        do {
            i++;
        } while (i < 10);
    }

    // || 运算符 +1, && +1 => 3
    public boolean logicOps(int a, int b) {
        return a > 0 || b > 0 && a < 100;
    }

    // lambda 表达式本身 base=1
    // (Runnable)() -> {} 复杂度 1
    public void lambdaSimple() {
        Runnable r = () -> System.out.println("hi");
    }

    // lambda 块内含 if +1, && +1 => 3
    public void lambdaWithLogic() {
        Runnable r = (x) -> {
            int v = (int) x;
            if (v > 0 && v < 100) {
                System.out.println(v);
            }
        };
    }

    // 多 catch 分支：base + try? + catch + catch => 3
    public void multiCatch() {
        try {
            Thread.sleep(1);
        } catch (InterruptedException e) {
            e.printStackTrace();
        } catch (RuntimeException e) {
            e.printStackTrace();
        }
    }

    // 嵌套三元：base + 2 个 conditional_expression => 3
    public int nestedTernary(int x) {
        return x > 0 ? (x > 10 ? 2 : 1) : 0;
    }

    // instanceof + 三元 + || => 4
    public int complexReturn(Object o) {
        if (o instanceof String s) {
            return s.length() > 0 || s.length() < 10 ? 1 : 2;
        }
        return 0;
    }
}
