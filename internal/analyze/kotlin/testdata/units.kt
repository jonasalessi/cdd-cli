package com.acme.app

class A                        // class
interface B                    // interface
enum class C { X, Y }          // enum (enum_entry: local_variable 0)
sealed class D                 // class
data class E(val v: Int)       // class (local_variable 1)
annotation class F             // class
object G                       // object
fun h() {}                     // function
fun String.shout() = uppercase()   // function, name "shout"
typealias I = (Int) -> Unit    // typealias
val j = { 1 }                  // property
val k: Int get() = 2           // property
val l by lazy { 3 }            // property (lambda inside `lazy { }` is +1: it is not the unit's own body)
private fun m() {}             // function (no visibility filter)
val n = 4                      // not a unit
class O {                      // class: one unit; everything below bills to it
    inner class P
    companion object {
        fun q() {}
    }
    object R
    fun s() { fun local() { if (x) {} } }
}
sealed interface S             // interface
fun interface Fn {             // interface
    fun call()
}
fun List<Map<String, Int>>.flat() = 1   // function, name "flat"
val cfg = load()               // not a unit
val s = "$n"                   // not a unit
val o = object : Runnable {    // not a unit
    override fun run() {}
}
val anon = fun(a: Int): Int = a   // property
var w: Int = 0                 // property
    set(value) { field = value }
internal class Q               // class
protected class T              // class (invalid outside a class, but the grammar parses it)
