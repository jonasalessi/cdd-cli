package com.acme.app;

import java.io.Closeable;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;

class Exceptions {
    void guard(Path p, Closeable existing) {
        try (var in = Files.newInputStream(p); existing) {
            // exception_handling 1 (the guarded block), local_variable 1 (in);
            // `existing` declares no name, so it is 0
            in.read();
        } catch (IOException | RuntimeException e) {
            // exception_handling 1 — multi-catch is one clause; `e` is 0
            throw new IllegalStateException(e);   // 0 — throw is neither branch nor handling
        }
    }
}
