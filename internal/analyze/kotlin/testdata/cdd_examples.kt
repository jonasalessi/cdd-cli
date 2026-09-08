// docs/cdd.md section 2, verbatim rules
fun check(a: Int, b: Int, c: Int, d: Int): Boolean {
    if (a > b && c < d) {     // code_branch 1, condition 2
        return true
    }
    return false
}                             // total 3

fun guarded() {
    try { a() } catch (e: E) { b() } finally { c() }   // exception_handling 3
}                             // total 3

fun ifElse(x: Int) = if (x > 0) 1 else 2               // code_branch 2 (if-else = 2)
