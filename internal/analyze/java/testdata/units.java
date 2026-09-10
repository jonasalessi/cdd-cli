package com.acme.app;

public class A {}                          // unit 1 — class, position at `class`
interface B {}                             // unit 2 — interface
enum C { X, Y }                            // unit 3 — enum
record D(int v) {}                         // unit 4 — record, local_variable 1 (component v)
@interface E {}                            // unit 5 — annotation, position at `@interface`
abstract class F {}                        // unit 6 — class, position at `class`, not `abstract`

final class G {                            // unit 7 — class; everything below bills to it
    static class Inner {}                  // not a unit
    class Nested {}                        // not a unit
    interface Shape {}                     // not a unit
    enum Kind { ONE }                      // not a unit
    record R(int v) {}                     // not a unit; local_variable 1 on G
    void m() {                             // not a unit — a member method
        class Local {}                     // not a unit
    }
}
