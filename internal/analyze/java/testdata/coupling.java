package com.acme.billing;

import com.acme.shared.Money;              // internal, binds Money
import com.acme.shared.Ledger;             // internal, binds Ledger
import static com.acme.shared.Rates.rate;  // internal, binds rate (the member)
import java.time.Instant;                  // stdlib, binds Instant
import java.util.*;                        // stdlib, star: charged to every unit

class Invoice {
    Money amount;                          // internal_coupling +1 (Money), local_variable 1
    Instant at() {                         // stdlib_coupling +1 (Instant)
        return Instant.ofEpochMilli(rate() + amount.cents());  // internal_coupling +1 (rate)
    }
}

class Note {
    Ledger l;                              // internal_coupling 1, local_variable 1
}

class Plain {
}
