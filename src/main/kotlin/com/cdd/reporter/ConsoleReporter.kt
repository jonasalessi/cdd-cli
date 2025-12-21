package com.cdd.reporter

import com.cdd.core.config.CddConfig
import com.cdd.core.config.OutputFormat
import com.cdd.domain.*

/**
 * Reporter that generates human-readable console output with ANSI colors.
 */
class ConsoleReporter : ReportGenerator {
    override val format = OutputFormat.CONSOLE

    private val BOLD = "\u001B[1m"
    private val RESET = "\u001B[0m"
    private val RED = "\u001B[31m"
    private val GREEN = "\u001B[32m"
    private val YELLOW = "\u001B[33m"
    private val CYAN = "\u001B[36m"

    override fun generate(analysis: AggregatedAnalysis, config: CddConfig): String {
        return buildString {
            appendLine("${BOLD}ICP Analysis Report${RESET}")
            appendLine("═".repeat(60))
            appendLine()

            appendLine("${BOLD}Summary${RESET}")
            appendLine("─".repeat(60))
            appendLine("Files analyzed:          ${analysis.totalFiles}")
            appendLine("Classes analyzed:        ${analysis.totalClasses}")
            appendLine("Total ICP:               ${String.format("%.1f", analysis.totalIcp)}")
            appendLine("Average ICP per class:   ${String.format("%.2f", analysis.averageIcp)}")
            
            val overLimitCount = analysis.classesOverLimit.size
            val overLimitPercent = if (analysis.totalClasses > 0) (overLimitCount.toDouble() / analysis.totalClasses) * 100 else 0.0
            appendLine("Classes over limit (>${config.limit}): ${if (overLimitCount > 0) RED else GREEN}$overLimitCount (${String.format("%.1f", overLimitPercent)}%)${RESET}")
            
            val largest = analysis.largestClasses.firstOrNull()
            if (largest != null) {
                appendLine("Largest class:           ${largest.name} (${String.format("%.1f", largest.totalIcp)} ICP)")
            }
            appendLine()

            appendLine("${BOLD}SLOC Metrics${RESET}")
            appendLine("─".repeat(60))
            appendLine("Average SLOC per class:  ${String.format("%.1f", analysis.slocMetrics.averageSlocPerClass)} (σ=${String.format("%.1f", analysis.slocMetrics.slocStdDev)})")
            appendLine("Average SLOC per method: ${String.format("%.1f", analysis.slocMetrics.averageSlocPerMethod)} (median=${analysis.slocMetrics.medianSlocPerMethod})")
            appendLine("Methods over 24 SLOC:    ${if (analysis.methodsOverSlocLimit.isNotEmpty()) YELLOW else GREEN}${analysis.methodsOverSlocLimit.size}${RESET}")
            appendLine("Total SLOC:              ${analysis.slocMetrics.totalSloc}")
            appendLine("Correlation (ICP vs SLOC): ${CYAN}${String.format("%.2f", analysis.icpSlocCorrelation)}${RESET}")
            appendLine()

            appendLine("${BOLD}SLOC Distribution${RESET}")
            appendLine("─".repeat(60))
            val maxCount = analysis.slocMetrics.slocDistribution.values.maxOrNull() ?: 1
            analysis.slocMetrics.slocDistribution.forEach { (bucket, count) ->
                val barLength = (count.toDouble() / maxCount * 40).toInt()
                val bar = "█".repeat(barLength)
                appendLine("${bucket.toString().padStart(4)}+: ${bar.padEnd(40)} ($count)")
            }
            appendLine()

            appendLine("${BOLD}ICP Distribution${RESET}")
            appendLine("─".repeat(60))
            analysis.icpDistribution.forEach { (type, count) ->
                val percent = if (analysis.totalIcp > 0) (count / analysis.totalIcp) * 100 else 0.0
                appendLine("${type.name.lowercase().replace('_', ' ').padEnd(20)}: ${count.toString().padEnd(5)} (${String.format("%.1f", percent)}%)")
            }
            appendLine()

            if (analysis.classesOverLimit.isNotEmpty()) {
                appendLine("${BOLD}Classes Over Limit${RESET}")
                appendLine("─".repeat(60))
                analysis.classesOverLimit.forEach { cls ->
                    appendLine("${RED}❌ ${cls.name}${RESET} (${String.format("%.1f", cls.totalIcp)} ICP)")
                    appendLine("   Package: ${cls.packageName}")
                    val breakdown = cls.icpBreakdown.map { (type, instances) -> 
                        "${type.name.lowercase().replace('_', ' ')}=${instances.size}"
                    }.joinToString(", ")
                    appendLine("   Breakdown: $breakdown")
                    appendLine()
                }
            }

            appendLine("${BOLD}Recommendations${RESET}")
            appendLine("─".repeat(60))
            if (analysis.suggestions.isEmpty()) {
                appendLine("• ✅ Everything looks good! Maintain the current complexity levels.")
            } else {
                analysis.suggestions.forEach { suggestion ->
                    appendLine("• $suggestion")
                }
            }
            appendLine()
            
            appendLine("${if (analysis.classesOverLimit.isEmpty()) GREEN else RED}${BOLD}Analysis complete. ${analysis.classesOverLimit.size} violations found.${RESET}")
        }
    }
}
