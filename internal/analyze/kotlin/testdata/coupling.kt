package com.acme.billing

import com.acme.shared.Money
import com.acme.shared.Ledger as L
import java.time.Instant             // stdlib: the JDK is Kotlin's other platform
import kotlinx.coroutines.*          // external: kotlinx ships apart from the language

class Invoice(val amount: Money) {    // internal 1 (Money), external 1 (star), stdlib 1 (Instant), local_variable 1
    fun at(): Instant = Instant.now()
}
class Note {                          // internal 1 (alias L -> Ledger), external 1 (star), stdlib 0, local_variable 1
    val l = L()
}
class Plain                           // internal 0, external 1 (star charges every unit), stdlib 0
