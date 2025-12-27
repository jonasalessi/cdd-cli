package com.cdd.reporter

import com.cdd.core.config.CddConfig
import com.cdd.core.config.OutputFormat
import com.cdd.domain.*
import kotlin.collections.component1
import kotlin.collections.component2

/**
 * Reporter that generates XML output.
 */
class XmlReporter : ReportGenerator {
    override val format = OutputFormat.XML

    override fun generate(analysis: AggregatedAnalysis, config: CddConfig): String {
        return """
            <?xml version="1.0" encoding="UTF-8"?>
            <report>
              <summary>
                <totalFiles>${analysis.totalFiles}</totalFiles>
                <totalClasses>${analysis.totalClasses}</totalClasses>
                <totalIcp>${analysis.totalIcp}</totalIcp>
                <averageIcp>${analysis.averageIcp}</averageIcp>
              </summary>
              <slocMetrics>
                <totalSloc>${analysis.slocMetrics.totalSloc}</totalSloc>
                <averageSlocPerClass>${analysis.slocMetrics.averageSlocPerClass}</averageSlocPerClass>
                <averageSlocPerMethod>${analysis.slocMetrics.averageSlocPerMethod}</averageSlocPerMethod>
                <icpSlocCorrelation>${analysis.icpSlocCorrelation}</icpSlocCorrelation>
              </slocMetrics>
              <icpDistribution>
                ${analysis.icpDistribution
                .map { (type, count) -> """<entry type="${type.name}">$count</entry>""" }
                .joinToString("\n")}
              </icpDistribution>
              <violations>
                ${analysis.classesOverLimit.joinToString("\n") { cls -> """
                <class name="${cls.name}" package="${cls.packageName}" icp="${cls.totalIcp}" />""".trimIndent() }}
              </violations>
              <suggestions>
                ${analysis.suggestions.joinToString("\n") { """
                <suggestion>$it</suggestion>""".trimIndent() }}
              </suggestions>
            </report>
        """.trimIndent()
    }

}
