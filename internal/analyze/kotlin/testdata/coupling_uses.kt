package com.acme.billing

import com.acme.shared.Money
import com.acme.shared.Money as M
import com.acme.shared.Ledger
import com.acme.shared.Same
import javax.inject.Inject
import com.acme.shared.Quoted
import com.acme.shared.Commented

class TypePosition {                  // internal 1 (Money, one module however many bindings)
    val m: Money = zero()
    val n = M.of(1)
}
class CallPosition {                  // internal 1 (Money via M)
    fun f() = M.of(1)
}
class Annotated {                     // external 1 (Inject)
    @Inject lateinit var x: String
}
class Untouched                       // 0: mentions nothing
class Shadowing {                     // internal 1: Ledger shadowed by a local still counts by name
    fun f() {
        val Ledger = 1
        println(Ledger)
    }
}
class SamePackage {                   // 0: SamePackageType needs no import and is invisible
    val s = SamePackageType()
}
class InString {                      // 0: "Quoted" and the comment are not identifiers
    val s = "Quoted" // Commented
}
