package com.cdd.reporter

import com.cdd.core.config.CddConfig
import com.cdd.core.config.OutputFormat
import com.cdd.domain.*

/**
 * Reporter that generates Markdown output.
 */
class MarkdownReporter : ReportGenerator {
    override val format = OutputFormat.MARKDOWN

    override fun generate(analysis: AggregatedAnalysis, config: CddConfig): String {
        return buildString {
            appendLine("# ICP Analysis Report")
            appendLine()
            appendLine("## Summary")
            appendLine()
            appendLine("| Metric | Value |")
            appendLine("| :--- | :--- |")
            appendLine("| Files analyzed | ${analysis.totalFiles} |")
            appendLine("| Classes analyzed | ${analysis.totalClasses} |")
            appendLine("| Total ICP | ${String.format("%.1f", analysis.totalIcp)} |")
            appendLine("| Average ICP per class | ${String.format("%.2f", analysis.averageIcp)} |")
            appendLine("| Classes over limit | ${analysis.classesOverLimit.size} |")
            appendLine()

            appendLine("## SLOC Metrics")
            appendLine()
            appendLine("| Metric | Value |")
            appendLine("| :--- | :--- |")
            appendLine("| Total SLOC | ${analysis.slocMetrics.totalSloc} |")
            appendLine("| Avg SLOC per class | ${String.format("%.1f", analysis.slocMetrics.averageSlocPerClass)} |")
            appendLine("| Avg SLOC per method | ${String.format("%.1f", analysis.slocMetrics.averageSlocPerMethod)} |")
            appendLine("| ICP-SLOC Correlation | ${String.format("%.2f", analysis.icpSlocCorrelation)} |")
            appendLine()

            appendLine("## SLOC Distribution")
            appendLine()
            appendLine("| Bucket (SLOC) | Count |")
            appendLine("| :--- | :---: |")
            analysis.slocMetrics.slocDistribution.forEach { (bucket, count) ->
                appendLine("| $bucket+ | $count |")
            }
            appendLine()

            appendLine("## ICP Distribution")
            appendLine()
            appendLine("| Type | Count | Percentage |")
            appendLine("| :--- | :---: | :---: |")
            analysis.icpDistribution.forEach { (type, count) ->
                val percent = if (analysis.totalIcp > 0) (count / analysis.totalIcp) * 100 else 0.0
                appendLine("| ${type.name.lowercase().replace('_', ' ')} | $count | ${String.format("%.1f", percent)}% |")
            }
            appendLine()

            if (analysis.classesOverLimit.isNotEmpty()) {
                appendLine("## Classes Over Limit")
                appendLine()
                appendLine("| Class | Package | ICP | Breakdown |")
                appendLine("| :--- | :--- | :---: | :--- |")
                analysis.classesOverLimit.forEach { cls ->
                    val breakdown = cls.icpBreakdown.map { (type, instances) -> 
                        "${type.name.lowercase().replace('_', ' ').take(3)}=${instances.size}"
                    }.joinToString(", ")
                    appendLine("| ${cls.name} | ${cls.packageName} | ${String.format("%.1f", cls.totalIcp)} | $breakdown |")
                }
                appendLine()
            }

            appendLine("## Recommendations")
            appendLine()
            if (analysis.suggestions.isEmpty()) {
                appendLine("- ✅ No issues found. All classes are within the complexity limit.")
            } else {
                analysis.suggestions.forEach { suggestion ->
                    appendLine("- $suggestion")
                }
            }
        }
    }
}
