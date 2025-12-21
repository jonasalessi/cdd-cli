package com.cdd.reporter

import com.cdd.core.config.*
import com.cdd.domain.*
import io.kotest.core.spec.style.FunSpec
import io.kotest.matchers.string.shouldContain
import java.io.File

class ReportGeneratorTest : FunSpec({

    val config = CddConfig(
        reporting = ReportingConfig(
            format = OutputFormat.CONSOLE
        )
    )

    val slocStats = SlocStatistics(
        totalSloc = 100,
        averageSlocPerClass = 50.0,
        averageSlocPerMethod = 10.0,
        medianSlocPerMethod = 8,
        slocStdDev = 5.0,
        slocDistribution = mapOf(0 to 2)
    )

    val aggregate = AggregatedAnalysis(
        totalFiles = 2,
        totalClasses = 2,
        totalIcp = 15.0,
        averageIcp = 7.5,
        classesOverLimit = listOf(
            ClassAnalysis(
                name = "HeavyClass",
                packageName = "com.example",
                lineRange = IntRangeSerializable(1, 100),
                totalIcp = 12.0,
                icpBreakdown = mapOf(IcpType.CODE_BRANCH to listOf(IcpInstance(IcpType.CODE_BRANCH, 5, 1, "if", 1.0))),
                methods = emptyList(),
                isOverLimit = true,
                sloc = SlocMetrics(100, 80, 90, 10, 10)
            )
        ),
        icpDistribution = mapOf(IcpType.CODE_BRANCH to 5, IcpType.CONDITION to 10),
        largestClasses = emptyList(),
        slocMetrics = slocStats,
        icpSlocCorrelation = 0.85,
        methodsOverSlocLimit = emptyList(),
        suggestions = listOf("Refactor HeavyClass")
    )

    test("JsonReporter should generate valid JSON") {
        val reporter = JsonReporter()
        val output = reporter.generate(aggregate, config)
        output shouldContain "\"totalFiles\": 2"
        output shouldContain "\"totalIcp\": 15.0"
        output shouldContain "\"HeavyClass\""
    }

    test("ConsoleReporter should generate human-readable output") {
        val reporter = ConsoleReporter()
        val output = reporter.generate(aggregate, config)
        output shouldContain "ICP Analysis Report"
        output shouldContain "Files analyzed:          2"
        output shouldContain "HeavyClass"
        output shouldContain "violations found"
    }

    test("XmlReporter should generate structured XML") {
        val reporter = XmlReporter()
        val output = reporter.generate(aggregate, config)
        output shouldContain "<?xml"
        output shouldContain "<totalFiles>2</totalFiles>"
        output shouldContain "<class name=\"HeavyClass\""
    }

    test("MarkdownReporter should generate markdown table") {
        val reporter = MarkdownReporter()
        val output = reporter.generate(aggregate, config)
        output shouldContain "# ICP Analysis Report"
        output shouldContain "| Files analyzed | 2 |"
        output shouldContain "## Classes Over Limit"
    }
})
