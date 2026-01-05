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
) {
    companion object {
        private val DEFAULT_METRICS = mapOf(
            "code_branch" to 1.0,
            "condition" to 1.0,
            "internal_coupling" to 1.0,
            "exception_handling" to 1.0
        )
        private val DEFAULT_LIMITS = mapOf(".*" to 12.0)

        val DEFAULT = CddConfig(
            metrics = mapOf(
                "java" to mapOf(".*" to DEFAULT_METRICS),
                "kotlin" to mapOf(".*" to DEFAULT_METRICS)
            ),
            icpLimits = mapOf(
                "java" to DEFAULT_LIMITS,
                "kotlin" to DEFAULT_LIMITS
            )
        )
    }

    fun mergeWith(other: CddConfig): CddConfig {
        return CddConfig(
            metrics = mergeMetrics(this.metrics, other.metrics),
            icpLimits = mergeLimits(this.icpLimits, other.icpLimits),
            internalCoupling = if (other.internalCoupling != InternalCouplingConfig()) other.internalCoupling else this.internalCoupling,
            include = if (other.include.isNotEmpty()) other.include else this.include,
            exclude = if (other.exclude.isNotEmpty()) other.exclude else this.exclude,
            sloc = if (other.sloc != SlocConfig()) other.sloc else this.sloc,
            reporting = if (other.reporting != ReportingConfig()) other.reporting else this.reporting
        )
    }

    private fun mergeMetrics(
        base: Map<String, Map<String, Map<String, Double>>>,
        other: Map<String, Map<String, Map<String, Double>>>
    ): Map<String, Map<String, Map<String, Double>>> {
        if (other.isEmpty()) return base
        val result = base.toMutableMap()
        other.forEach { (lang, patterns) ->
            val langPatterns = result[lang]?.toMutableMap() ?: mutableMapOf()
            patterns.forEach { (pattern, weights) ->
                val patternWeights = langPatterns[pattern]?.toMutableMap() ?: mutableMapOf()
                patternWeights.putAll(weights)
                langPatterns[pattern] = patternWeights
            }
            result[lang] = langPatterns
        }
        return result
    }

    private fun mergeLimits(
        base: Map<String, Map<String, Double>>,
        other: Map<String, Map<String, Double>>
    ): Map<String, Map<String, Double>> {
        if (other.isEmpty()) return base
        val result = base.toMutableMap()
        other.forEach { (lang, patterns) ->
            val langPatterns = result[lang]?.toMutableMap() ?: mutableMapOf()
            langPatterns.putAll(patterns)
            result[lang] = langPatterns
        }
        return result
    }
}
