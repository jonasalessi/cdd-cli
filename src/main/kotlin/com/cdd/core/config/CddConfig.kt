package com.cdd.core.config

import com.cdd.domain.IcpType
import kotlinx.serialization.Serializable

/**
 * Configuration for Output Formats.
 */
@Serializable
enum class OutputFormat {
    CONSOLE, JSON, XML, MARKDOWN
}

/**
 * Configuration for Internal Coupling detection.
 */
@Serializable
data class InternalCouplingConfig(
    val autoDetect: Boolean = true,
    val packages: List<String> = emptyList()
)

/**
 * Configuration for External Coupling detection.
 */
@Serializable
data class ExternalCouplingConfig(
    val libraries: List<String> = listOf(
        "java.util.*",
        "java.io.*",
        "java.lang.*"
    )
)

/**
 * Configuration for SLOC (Source Lines of Code) metrics.
 */
@Serializable
data class SlocConfig(
    val classLimit: Int = 0,
    val methodLimit: Int = 24,
    val warnAtMethod: Int = 15,
    val excludeComments: Boolean = true,
    val excludeBlankLines: Boolean = true
)

/**
 * Configuration for reporting options.
 */
@Serializable
data class ReportingConfig(
    val format: OutputFormat = OutputFormat.CONSOLE,
    val outputFile: String? = null,
    val verbose: Boolean = false,
    val showLineNumbers: Boolean = true,
    val showSuggestions: Boolean = true,
    val showSlocMetrics: Boolean = true,
    val showSlocDistribution: Boolean = true,
    val showCorrelation: Boolean = true
)

/**
 * Main CDD Configuration model.
 */
@Serializable
data class CddConfig(
    val limit: Double = 10.0,
    val icpTypes: Map<IcpType, Double> = IcpType.entries.associateWith { it.defaultWeight },
    val classTypeLimits: Map<String, Int> = emptyMap(),
    val internalCoupling: InternalCouplingConfig = InternalCouplingConfig(),
    val externalCoupling: ExternalCouplingConfig = ExternalCouplingConfig(),
    val include: List<String> = emptyList(),
    val exclude: List<String> = emptyList(),
    val sloc: SlocConfig = SlocConfig(),
    val reporting: ReportingConfig = ReportingConfig()
)
