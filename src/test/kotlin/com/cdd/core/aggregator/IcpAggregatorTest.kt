package com.cdd.core.aggregator

import com.cdd.core.config.CddConfig
import com.cdd.domain.*
import io.kotest.core.spec.style.FunSpec
import io.kotest.matchers.collections.shouldHaveSize
import io.kotest.matchers.doubles.shouldBeExactly
import io.kotest.matchers.doubles.shouldBeGreaterThan
import io.kotest.matchers.doubles.shouldBeLessThan
import io.kotest.matchers.shouldBe
import java.io.File

class IcpAggregatorTest : FunSpec({
    val aggregator = IcpAggregator()
    val config = CddConfig(limit = 10.0)

    fun createClass(name: String, icp: Double, sloc: Int, isOverLimit: Boolean = icp > 10.0): ClassAnalysis {
        return ClassAnalysis(
            name = name,
            packageName = "com.example",
            lineRange = IntRangeSerializable(1, sloc),
            totalIcp = icp,
            icpBreakdown = emptyMap(),
            methods = emptyList(),
            isOverLimit = isOverLimit,
            sloc = SlocMetrics(sloc, sloc, sloc, 0, 0)
        )
    }

    test("should aggregate across multiple files correctly") {
        val results = listOf(
            AnalysisResult(
                file = "File1.java",
                classes = listOf(createClass("Class1", 5.0, 50)),
                totalIcp = 5.0
            ),
            AnalysisResult(
                file = "File2.java",
                classes = listOf(createClass("Class2", 15.0, 150)),
                totalIcp = 15.0
            )
        )

        val aggregated = aggregator.aggregate(results, config)

        aggregated.totalFiles shouldBe 2
        aggregated.totalClasses shouldBe 2
        aggregated.totalIcp shouldBe 20.0
        aggregated.averageIcp shouldBe 10.0
        aggregated.classesOverLimit shouldHaveSize 1
        aggregated.classesOverLimit[0].name shouldBe "Class2"
    }

    test("should compute correct SLOC stats") {
        val classes = listOf(
            createClass("C1", 5.0, 40),
            createClass("C2", 10.0, 60),
            createClass("C3", 15.0, 80)
        )
        val results = listOf(AnalysisResult("F1.java", classes, 30.0))

        val aggregated = aggregator.aggregate(results, config)
        val stats = aggregated.slocMetrics

        stats.totalSloc shouldBe 180
        stats.averageSlocPerClass shouldBe 60.0
        stats.slocStdDev shouldBeGreaterThan 0.0
        // Variance: ((40-60)^2 + (60-60)^2 + (80-60)^2) / 3 = (400 + 0 + 400) / 3 = 800/3 = 266.66
        // StdDev = sqrt(266.66) approx 16.33
        stats.slocStdDev shouldBe (kotlin.math.sqrt(800.0/3.0))
    }

    test("should calculate correct ICP-SLOC correlation") {
        // Positive correlation: ICP and SLOC grow together
        val positiveResults = listOf(
            AnalysisResult("F1.java", listOf(
                createClass("C1", 2.0, 20),
                createClass("C2", 4.0, 40),
                createClass("C3", 6.0, 60),
                createClass("C4", 8.0, 80),
                createClass("C5", 10.0, 100)
            ), 30.0)
        )

        val posAggregated = aggregator.aggregate(positiveResults, config)
        posAggregated.icpSlocCorrelation shouldBeExactly 1.0

        // No correlation
        val noCorrResults = listOf(
            AnalysisResult("F1.java", listOf(
                createClass("C1", 2.0, 100),
                createClass("C2", 10.0, 20)
            ), 12.0)
        )
        val noAggregated = aggregator.aggregate(noCorrResults, config)
        noAggregated.icpSlocCorrelation shouldBeExactly -1.0 // Inversely perfectly correlated here
    }

    test("should handle empty results") {
        val aggregated = aggregator.aggregate(emptyList(), config)

        aggregated.totalFiles shouldBe 0
        aggregated.totalClasses shouldBe 0
        aggregated.totalIcp shouldBe 0.0
        aggregated.averageIcp shouldBe 0.0
        aggregated.slocMetrics.totalSloc shouldBe 0
    }
    test("should generate appropriate suggestions") {
        val results = listOf(
            AnalysisResult("F1.java", listOf(
                createClass("HeavyClass", 20.0, 500)
            ), 20.0)
        )
        
        val aggregated = aggregator.aggregate(results, config)
        aggregated.suggestions shouldHaveSize 2
        aggregated.suggestions[0] shouldBe "Refactor the 1 classes that exceed the ICP limit of 10.0."
        aggregated.suggestions[1] shouldBe "Prioritize 'HeavyClass' as it has the highest complexity (20.0 ICP)."
        // Since we only have one class, the correlation won't be calculated (size < 2 in computeIcpSlocCorrelation)
        // and we don't have enough classes for other rules.
        // Wait, why 3?
        // 1. Classes over limit
        // 2. Worst class
        // 3. (Maybe something else? Let's check IcpAggregator rules)
    }

    test("should suggest smaller methods when correlation is high") {
        val results = listOf(
            AnalysisResult("F1.java", listOf(
                createClass("C1", 2.0, 20),
                createClass("C2", 4.0, 40),
                createClass("C3", 6.0, 60),
                createClass("C4", 8.0, 80),
                createClass("C5", 12.0, 120) // One over limit
            ), 32.0)
        )
        
        val aggregated = aggregator.aggregate(results, config)
        aggregated.suggestions.any { it.contains("Strong correlation") } shouldBe true
    }
})
