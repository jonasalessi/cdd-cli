package com.acme.app;

class Locals {
    private int a, b;                      // local_variable 2 — one per declarator
    static final int MAX = 3;              // local_variable 1

    void body(Object o) {
        int x = 1, y = 2;                  // local_variable 2
        var z = x;                         // local_variable 1
        if (o instanceof String s) {       // code_branch 1, local_variable 0 (pattern variable)
            z = s.length();
        }
        try {                              // exception_handling 1
            check(y + z + MAX + a + b);
        } catch (Exception e) {            // exception_handling 1, local_variable 0 (catch param)
            return;
        }
    }

    private void check(int v) {
    }
}

interface Consts {
    int LIMIT = 10;                        // local_variable 1 — a constant_declaration
}

enum Color {
    RED, GREEN                             // local_variable 0 — enum constants are not variables
}
