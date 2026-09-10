package com.acme.app;

class Conditions {
    boolean both(boolean a, boolean b) {
        return a && b;                     // condition 2
    }

    boolean either(boolean a, boolean b, boolean c) {
        return a && b || c;                // condition 3
    }

    boolean negated(boolean a, boolean b, boolean x) {
        return !(a || b) && x;             // condition 3 — flattened through ! and ( )
    }

    boolean compare(int x) {
        return x > 1;                      // condition 0 — a comparison is not a clause
    }

    int bits(int a, int b) {
        return a & b | 3;                  // condition 0 — bitwise operators are not clauses
    }
}
