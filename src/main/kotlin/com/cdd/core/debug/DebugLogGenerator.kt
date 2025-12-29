package com.cdd.core.debug

import com.cdd.core.registry.AnalyzerRegistry
import com.cdd.domain.AnalysisResult
import com.cdd.domain.IcpInstance
import com.cdd.domain.IcpType
import java.io.File
import java.nio.file.Files
import java.nio.file.Path
import java.nio.file.Paths
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter
import kotlin.io.path.appendText
import kotlin.io.path.writeText

/**
 * Generates debug log files containing source code annotated with ICP (Intrinsic Cognitive Complexity) metrics.
 * 
 * The generator processes analysis results and creates a log file where each line of code is annotated
 * with its associated ICP instances, showing the complexity breakdown by type (branches, conditions, etc.).
 */
object DebugLogGenerator {
    private val DEFAULT_LOG_FILENAME = "cdd-debug-${LocalDateTime.now().format(DateTimeFormatter.ofPattern("yyyyMMdd-HHmmss"))}.log"
    private const val FILE_SEPARATOR = "------"
    private const val ICP_COMMENT_PREFIX = " //ICP= "

    /**
     * Generates a debug log file from the given analysis results.
     * 
     * @param results List of analysis results containing ICP metrics for analyzed files
     * @param logPath Optional path where the log file should be created. Defaults to current directory.
     */
    fun generate(results: List<AnalysisResult>, logPath: String?) {
        val logFile = createLogFile(logPath)
        logFile.writeText("")

        results.forEach { result ->
            processAnalysisResult(result, logFile)
        }
    }

    /**
     * Processes a single analysis result and appends the annotated source code to the log file.
     */
    private fun processAnalysisResult(result: AnalysisResult, logFile: Path) {
        val file = File(result.file)

        if (AnalyzerRegistry.getAnalyzerFor(file) == null) return

        logFile.appendText("File: ${result.file}\n")

        val icpByLine = extractIcpInstancesByLine(result)
        appendAnnotatedSourceCode(file, icpByLine, logFile)
        logFile.appendText("$FILE_SEPARATOR\n")
    }

    private fun extractIcpInstancesByLine(result: AnalysisResult): Map<Int, List<IcpInstance>> {
        return result.classes
            .flatMap { it.icpBreakdown.values.flatten() }
            .groupBy { it.line }
    }

    private fun appendAnnotatedSourceCode(
        file: File,
        icpByLine: Map<Int, List<IcpInstance>>,
        logFile: Path
    ) {
        file.readLines().forEachIndexed { index, line ->
            val lineNumber = index + 1
            val lineIcps = icpByLine[lineNumber]

            val annotatedLine = if (lineIcps.isNullOrEmpty()) {
                line
            } else {
                val summary = formatIcpSummary(lineIcps)
                "$line$ICP_COMMENT_PREFIX$summary"
            }

            logFile.appendText("$annotatedLine\n")
        }
    }

    /**
     * Formats a list of ICP instances into a human-readable summary string.
     * 
     * Example output: "+2 Branches, +1.5 Condition, +1 Exception"
     */
    private fun formatIcpSummary(icpInstances: List<IcpInstance>): String {
        return icpInstances
            .groupBy { it.type }
            .map { (type, instances) ->
                val totalWeight = instances.sumOf { it.weight }
                val count = instances.size
                val typeLabel = getTypeLabelForDisplay(type)
                val weightLabel = formatWeight(totalWeight)
                val pluralizedLabel = pluralize(typeLabel, count)

                "+$weightLabel $pluralizedLabel"
            }
            .joinToString(", ")
    }

    private fun getTypeLabelForDisplay(type: IcpType): String = when (type) {
        IcpType.CODE_BRANCH -> "Branch"
        IcpType.CONDITION -> "Condition"
        IcpType.EXCEPTION_HANDLING -> "Exception"
        IcpType.INTERNAL_COUPLING -> "Internal"
        IcpType.EXTERNAL_COUPLING -> "External"
    }

    /**
     * Formats a weight value, removing unnecessary decimal points for whole numbers.
     * 
     * Examples: 2.0 -> "2", 1.5 -> "1.5"
     */
    private fun formatWeight(weight: Double): String {
        return if (weight == weight.toInt().toDouble()) {
            weight.toInt().toString()
        } else {
            weight.toString()
        }
    }

    private fun pluralize(word: String, count: Int): String {
        return if (count > 1) "${word}s" else word
    }

    private fun createLogFile(logPath: String?): Path {
        val directory = if (logPath.isNullOrBlank()) {
            Paths.get(".")
        } else {
            Paths.get(logPath)
        }

        Files.createDirectories(directory)
        return directory.resolve(DEFAULT_LOG_FILENAME)
    }
}
