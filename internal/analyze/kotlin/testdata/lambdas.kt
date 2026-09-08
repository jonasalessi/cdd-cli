fun total(xs: List<Int>): Int {
    val doubled = xs.map { it * 2 }        // lambda 1, local_variable 1
    val shown = xs.map(::render)           // lambda 1, local_variable 1
    val anon = fun(a: Int): Int = a        // lambda 1, local_variable 1
    return xs.let { it.sum() }             // lambda 1
}                                          // lambda 4, local_variable 3

val onEvent: (Int) -> Unit = { println(it) }   // unit "onEvent" kind property; lambda 0 (FR-11), local_variable 0
val plain = 3                                  // not a unit

fun scopes(x: String) {
    x.let { }                              // lambda 1
    x.apply { }                            // lambda 1
    x.also { }                             // lambda 1
    x.run { }                              // lambda 1
    with(x) { }                            // lambda 1
    x.takeIf { true }                      // lambda 1
}                                          // lambda 6

val lazyValue by lazy { 3 }                // unit; lambda 1

fun nested(xs: List<List<Int>>) = xs.map { it.filter { it > 0 } }   // lambda 2

fun references(s: String) {
    val a = String::trim                   // 0: parses as navigation_expression in v1.1.0
    val b = ::render                       // lambda 1
    val c = this::render                   // see the test
    val d = Foo::class                     // see the test
}

class WithLambda {
    val f = { 1 }                          // lambda 1, local_variable 1
}

fun coroutines() {
    launch { }                             // lambda 1
    withContext(Dispatchers.IO) { }        // lambda 1
    runCatching { }                        // lambda 1, exception_handling 0
}                                          // lambda 3
