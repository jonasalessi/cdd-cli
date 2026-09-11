package com.acme.app;

class Ledger extends Base implements Auditable, Printer {
    // inheritance 3 — Base, Auditable, Printer, one occurrence each
}

interface Auditable extends Named, Timestamped {
    // inheritance 2
}

enum Level implements Auditable {
    LOW, HIGH                              // inheritance 1, local_variable 0 (enum constants)
}

record Money(int amount, String currency) implements Comparable<Money> {
    // inheritance 1 (type arguments do not add), local_variable 2 (the components)
    public int compareTo(Money other) {
        return amount - other.amount;
    }
}

class Factory {
    Runnable r = new Runnable() {          // inheritance 1 (anonymous class, on `Runnable`),
        public void run() {                // local_variable 1 (the field r), lambda 0
        }
    };
}

sealed interface Shape permits Circle {
    // inheritance 0 — `permits` is not an edge; Circle lives elsewhere in the package
}
