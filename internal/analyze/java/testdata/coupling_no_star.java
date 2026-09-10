package com.acme.billing;

import com.acme.shared.Money;
import com.acme.shared.Ledger;
import static com.acme.shared.Rates.rate;
import java.time.Instant;

class Invoice {
    Money amount;
    Instant at() {
        return Instant.ofEpochMilli(rate() + amount.cents());
    }
}

class Note {
    Ledger l;
}

class Plain {
}
