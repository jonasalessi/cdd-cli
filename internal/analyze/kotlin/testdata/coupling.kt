package com.acme.billing

import com.acme.shared.Money
import com.acme.shared.Ledger as L
import java.time.Instant
import kotlinx.coroutines.*

class Invoice(val amount: Money) {    // internal 1 (Money), external 2 (Instant, star), local_variable 1
    fun at(): Instant = Instant.now()
}
class Note {                          // internal 1 (alias L -> Ledger), external 1 (star), local_variable 1
    val l = L()
}
class Plain                           // internal 0, external 1 (star charges every unit)
