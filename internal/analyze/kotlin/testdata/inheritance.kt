class Ledger(private val repo: Repo, clock: Clock) : Base(), Auditable, Printer by ConsolePrinter()
// inheritance 3, local_variable 1 (repo; clock is a plain parameter)

interface Auditable : Named, Timestamped                 // inheritance 2
object Registry : Auditable                              // inheritance 1
class Bare                                               // inheritance 0
class Outer {
    inner class In : Base()                              // inheritance 1, on Outer
}
enum class Color : Named {                               // inheritance 1
    RED
}
class X : Generic<String>()                              // inheritance 1
fun anonymous() {
    val r = object : Runnable {                          // inheritance 1, local_variable 1
        override fun run() {}
    }
}
