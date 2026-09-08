class P(val a: Int, var b: Int, c: Int)                  // local_variable 2

fun body(pair: Pair<Int, Int>) {
    val x = 1                                            // local_variable 1
    var y = 2                                            // local_variable 1
    val (p, q) = pair                                    // local_variable 1
    lateinit var z: String                               // local_variable 1
}                                                        // local_variable 4

class Members {
    val a = 1                                            // local_variable 1
    val b: Int get() = 2                                 // local_variable 1
    val c by lazy { 3 }                                  // local_variable 1, lambda 1
}                                                        // local_variable 3

fun loops(xs: List<Int>, m: Map<String, Int>) {
    for (x in xs) { }                                    // local_variable 1
    for ((k, v) in m) { }                                // local_variable 1
}                                                        // local_variable 2

interface Shape {
    val x: Int                                           // 0: a shape
    val y: Int get() = 1                                 // local_variable 1: an accessor body
    fun f()
}                                                        // local_variable 1

abstract class Abstract {
    abstract val x: Int                                  // local_variable 1: the exemption is for interfaces only
}

enum class Colors { X, Y, Z }                            // local_variable 0

fun params(a: Int, b: Int) {
    val f = { p: Int, q: Int -> p + q }                  // local_variable 1, lambda 1
    try { a() } catch (e: E) { b() }                     // catch binding: 0
    xs.map { it }                                        // it: 0
}                                                        // local_variable 1

val j = { 1 }                                            // unit j: local_variable 0

class Companion {
    companion object {
        val inside = 1                                   // local_variable 1
    }
    init {
        val scoped = 2                                   // local_variable 1
    }
}                                                        // local_variable 2
