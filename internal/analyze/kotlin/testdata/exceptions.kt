fun twoCatches() {
    try {
        a()
    } catch (e: E) {
        b()
    } catch (f: F) {
        c()
    }
}                                          // exception_handling 3

fun finallyOnly() {
    try {
        a()
    } finally {
        c()
    }
}                                          // exception_handling 2

fun caught() {
    runCatching { a() }                    // exception_handling 0, lambda 1
}
