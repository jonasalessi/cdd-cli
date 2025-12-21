package com.cdd.reporter

import com.cdd.core.config.CddConfig
import com.cdd.domain.AggregatedAnalysis
import com.cdd.core.config.OutputFormat
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json

/**
 * Reporter that generates JSON output.
 */
class JsonReporter : ReportGenerator {
    override val format = OutputFormat.JSON

    private val json = Json {
        prettyPrint = true
        ignoreUnknownKeys = true
    }

    override fun generate(analysis: AggregatedAnalysis, config: CddConfig): String {
        return json.encodeToString(analysis)
    }
}
