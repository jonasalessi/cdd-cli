package com.cdd.core.config

import com.cdd.domain.IcpType
import kotlinx.serialization.SerialName
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
    @SerialName("auto_detect")
    val autoDetect: Boolean = true,
    val packages: List<String> = emptyList()
)


/**
 * Configuration for SLOC (Source Lines of Code) metrics.
 */
@Serializable
data class SlocConfig(
    val methodLimit: Int = 24
)

/**
 * Configuration for reporting options.
 */
@Serializable
data class ReportingConfig(
    val format: OutputFormat = OutputFormat.CONSOLE,
    val outputFile: String? = null
)

/**
 * Main CDD Configuration model.
 */
@Serializable
data class CddConfig(
    // Language -> Regex -> Metric -> Weight
    val metrics: Map<String, Map<String, Map<String, Double>>> = emptyMap(),
    @SerialName("icp-limits")
    val icpLimits: Map<String, Map<String, Double>> = emptyMap(),
    @SerialName("internal_coupling")
    val internalCoupling: InternalCouplingConfig = InternalCouplingConfig(),
    val include: List<String> = emptyList(),
    val exclude: List<String> = emptyList(),
    val sloc: SlocConfig = SlocConfig(),
    @SerialName("reporter")
    val reporting: ReportingConfig = ReportingConfig()
)
