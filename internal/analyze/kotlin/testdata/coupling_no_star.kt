package com.acme.billing

import com.acme.shared.Money
import com.acme.shared.Ledger as L
import java.time.Instant

class Invoice(val amount: Money) {    // internal 1, external 1
    fun at(): Instant = Instant.now()
}
class Note {                          // internal 1, external 0
    val l = L()
}
class Plain                           // internal 0, external 0
