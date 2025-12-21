package com.sample

class SimpleKotlin {
    fun sum(a: Int, b: Int): Int {
        if (a > 0 && b > 0) { // +1 branch, +2 conditions
            return a + b
        }
        return 0
    }
}
