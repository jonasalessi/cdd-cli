package com.cdd.core.registry

import com.cdd.analyzer.LanguageAnalyzer
import java.io.File
import org.slf4j.LoggerFactory

/**
 * Manages registration and discovery of language analyzers.
 */
object AnalyzerRegistry {
    private val logger = LoggerFactory.getLogger(AnalyzerRegistry::class.java)
    private val analyzers = mutableListOf<LanguageAnalyzer>()

    /**
     * Registers a new language analyzer.
     */
    fun register(analyzer: LanguageAnalyzer) {
        logger.info("Registering analyzer: ${analyzer.languageName} for extensions: ${analyzer.supportedExtensions}")
        analyzers.add(analyzer)
    }

    /**
     * Returns the appropriate analyzer for a given file, or null if unsupported.
     */
    fun getAnalyzerFor(file: File): LanguageAnalyzer? {
        val extension = file.extension.lowercase()
        val analyzer = analyzers.find { it.supportedExtensions.contains(extension) }
        
        if (analyzer == null) {
            logger.warn("No analyzer found for file: ${file.name} (extension: .$extension)")
        }
        
        return analyzer
    }

    /**
     * Clears all registered analyzers (primarily for testing).
     */
    fun clear() {
        analyzers.clear()
    }
}
