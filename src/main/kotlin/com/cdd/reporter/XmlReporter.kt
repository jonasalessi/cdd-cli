package com.cdd.reporter

import com.cdd.core.config.CddConfig
import com.cdd.core.config.OutputFormat
import com.cdd.domain.*

/**
 * Reporter that generates XML output.
 */
class XmlReporter : ReportGenerator {
    override val format = OutputFormat.XML

    override fun generate(analysis: AggregatedAnalysis, config: CddConfig): String {
        val sb = StringBuilder()
        sb.append("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
        sb.append("<report>\n")
        
        sb.append("  <summary>\n")
        sb.append("    <totalFiles>${analysis.totalFiles}</totalFiles>\n")
        sb.append("    <totalClasses>${analysis.totalClasses}</totalClasses>\n")
        sb.append("    <totalIcp>${analysis.totalIcp}</totalIcp>\n")
        sb.append("    <averageIcp>${analysis.averageIcp}</averageIcp>\n")
        sb.append("  </summary>\n")

        sb.append("  <slocMetrics>\n")
        sb.append("    <totalSloc>${analysis.slocMetrics.totalSloc}</totalSloc>\n")
        sb.append("    <averageSlocPerClass>${analysis.slocMetrics.averageSlocPerClass}</averageSlocPerClass>\n")
        sb.append("    <averageSlocPerMethod>${analysis.slocMetrics.averageSlocPerMethod}</averageSlocPerMethod>\n")
        sb.append("    <icpSlocCorrelation>${analysis.icpSlocCorrelation}</icpSlocCorrelation>\n")
        sb.append("  </slocMetrics>\n")

        sb.append("  <icpDistribution>\n")
        analysis.icpDistribution.forEach { (type, count) ->
            sb.append("    <entry type=\"${type.name}\">$count</entry>\n")
        }
        sb.append("  </icpDistribution>\n")

        sb.append("  <violations>\n")
        analysis.classesOverLimit.forEach { cls ->
            sb.append("    <class name=\"${cls.name}\" package=\"${cls.packageName}\" icp=\"${cls.totalIcp}\" />\n")
        }
        sb.append("  </violations>\n")

        sb.append("  <suggestions>\n")
        analysis.suggestions.forEach { suggestion ->
            sb.append("    <suggestion>$suggestion</suggestion>\n")
        }
        sb.append("  </suggestions>\n")

        sb.append("</report>")
        return sb.toString()
    }
}
