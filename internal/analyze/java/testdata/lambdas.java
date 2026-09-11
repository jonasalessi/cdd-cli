package com.acme.app;

import java.util.ArrayList;
import java.util.List;
import java.util.function.Function;
import java.util.function.IntUnaryOperator;
import java.util.function.Supplier;

class Lambdas {
    void wire() {
        IntUnaryOperator doubled = x -> x * 2;              // lambda 1, local_variable 1
        Function<Integer, String> shown = String::valueOf;  // lambda 1, local_variable 1
        Supplier<List<String>> made = ArrayList::new;       // lambda 1, local_variable 1
        Runnable noop = () -> {};                           // lambda 1, local_variable 1
        run(doubled, shown, made, noop);
    }
}
