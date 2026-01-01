package com.cdd.analyzer

import com.cdd.core.config.CddConfig
import java.io.File

/**
 * Base class for all language analyzers, providing shared functionality.
 */
abstract class AbstractLanguageAnalyzer : LanguageAnalyzer {
    
    protected fun resolveWeights(file: File, config: CddConfig): Map<String, Double> {
        val langKey = languageName.lowercase()
        val languageMetrics = config.metrics[langKey] ?: return emptyMap()

        for ((pattern, weights) in languageMetrics) {
            try {
                val regex = pattern.toRegex()
                if (regex.matches(file.name) || regex.matches(file.absolutePath)) {
                    return weights
                }
            } catch (e: Exception) {
                // Ignore invalid regex
            }
        }
        return emptyMap()
    }

    protected fun resolveIcpLimit(file: File, config: CddConfig): Double? {
        val langKey = languageName.lowercase()
        val languageLimits = config.icpLimits[langKey] ?: return null

        for ((pattern, limit) in languageLimits) {
            try {
                val regex = pattern.toRegex()
                if (regex.matches(file.name) || regex.matches(file.absolutePath)) {
                    return limit
                }
            } catch (e: Exception) {
                // Ignore
            }
        }
        return null
    }
}
