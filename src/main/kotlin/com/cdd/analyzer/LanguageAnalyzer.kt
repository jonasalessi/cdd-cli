package com.cdd.analyzer

import com.cdd.core.config.CddConfig
import com.cdd.domain.AnalysisResult
import java.io.File

/**
 * Common interface for all language-specific analyzers.
 */
interface LanguageAnalyzer {
    /**
     * Supported file extensions (e.g., ["java", "kt"]).
     */
    val supportedExtensions: List<String>

    /**
     * Human-readable language name (e.g., "Java", "Kotlin").
     */
    val languageName: String

    /**
     * Analyzes a source file and returns the analysis result.
     */
    fun analyze(file: File, config: CddConfig): AnalysisResult

    /**
     * Strips comments from a single line of code.
     */
    fun stripComments(line: String): String
}
