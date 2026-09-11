package com.acme.app;

import java.util.List;

class Branches {
    String oldSwitch(int x) {
        switch (x) {
            case 1:                        // code_branch 1 — an arm of one label
                return "one";
            case 2:                        // falls through into case 3
            case 3:                        // code_branch 1 — one arm, two labels
                return "few";
            default:                       // 0
                return "many";
        }
    }

    String arrowSwitch(int x) {
        return switch (x) {
            case 1 -> "one";               // code_branch 1
            case 2, 3 -> "few";            // code_branch 1 — one rule
            default -> "many";             // 0
        };
    }

    int chain(boolean a, boolean b, boolean c) {
        if (a) {                           // code_branch 1
            return 1;
        } else if (b) {                    // code_branch 1 — the if; its else is an if, so 0
            return 2;
        } else if (c) {                    // code_branch 1
            return 3;
        } else {                           // code_branch 1 — the final alternative
            return 4;
        }
    }

    int ternary(int x) {
        return x > 0 ? 1 : 2;              // code_branch 1
    }

    void loops(List<Integer> xs, boolean flag) {
        for (int i = 0; i < 3; i++) {      // code_branch 1, local_variable 1 (i)
            noop();
        }
        for (Integer x : xs) {             // code_branch 1, local_variable 1 (x)
            noop();
        }
        while (flag) {                     // code_branch 1
            noop();
        }
        do {                               // code_branch 1
            noop();
        } while (flag);
    }
}
