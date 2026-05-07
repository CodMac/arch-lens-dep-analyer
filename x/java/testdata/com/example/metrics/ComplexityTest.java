package com.example.metrics;

public class ComplexityTest {
    // Complexity = 1 (base)
    public void simple() {}

    // Complexity = 1 (base) + 1 (if) + 1 (for) + 1 (&&) = 4
    public void medium(int x) {
        if (x > 0 && x < 10) {
            for (int i = 0; i < x; i++) {
                System.out.println(i);
            }
        }
    }

    // Complexity = 1 (base) + 3 (case/default) = 4
    public void complexSwitch(int x) {
        switch(x) {
            case 1: 
                System.out.println("1");
                break;
            case 2: 
                System.out.println("2");
                break;
            default: 
                System.out.println("def");
                break;
        }
    }

    // Complexity = 1 (base) + 1 (while) + 1 (catch) = 3
    public void withTryCatch() {
        try {
            while(true) {
                throw new Exception();
            }
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    // Complexity = 1 (base) + 1 (ternary) = 2
    public int ternary(int x) {
        return x > 0 ? 1 : 0;
    }

    // Complexity = 1 (base) + 3 (switch_rule) = 4
    public void arrowSwitch(int x) {
        switch (x) {
            case 1 -> System.out.println("1");
            case 2 -> System.out.println("2");
            default -> System.out.println("def");
        }
    }
}
