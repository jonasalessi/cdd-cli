package com.acme.app;

class Examples {
    boolean check(int a, int b, int c, int d) {
        if (a > b && c < d) {              // code_branch 1, condition 2
            return true;
        }
        return false;
    }

    void guarded() {
        try {                              // exception_handling 1 (on the guarded block)
            first();
        } catch (RuntimeException e) {     // exception_handling 1
            second();
        } finally {                        // exception_handling 1
            third();
        }
    }

    int ifElse(int x) {
        if (x > 0) {                       // code_branch 1
            return 1;
        } else {                           // code_branch 1 — the alternative is not an if
            return 2;
        }
    }
}
