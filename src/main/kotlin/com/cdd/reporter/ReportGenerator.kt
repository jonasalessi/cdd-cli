package com.cdd.reporter

import com.cdd.core.config.CddConfig
import com.cdd.domain.AggregatedAnalysis
import com.cdd.core.config.OutputFormat

/**
 * Interface for generating reports in various formats.
 */
interface ReportGenerator {
    /**
     * The output format supported by this generator.
     */
    val format: OutputFormat

    /**
     * Generates a report based on the aggregated analysis.
     * @return The formatted report as a string.
     */
    fun generate(analysis: AggregatedAnalysis, config: CddConfig): String
}

/**
 * Registry to manage and retrieve reporters by format.
 */
object ReporterRegistry {
    private val reporters = mutableMapOf<OutputFormat, ReportGenerator>()

    fun register(reporter: ReportGenerator) {
        reporters[reporter.format] = reporter
    }

    fun getReporter(format: OutputFormat): ReportGenerator {
        return reporters[format] ?: throw IllegalArgumentException("No reporter found for format: $format")
    }
}
