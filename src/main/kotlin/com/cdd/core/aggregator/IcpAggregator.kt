package com.cdd.core.aggregator

import com.cdd.core.config.CddConfig
import com.cdd.domain.*
import kotlin.math.sqrt

/**
 * Aggregates results from multiple language analyzers and computes global statistics.
 */
class IcpAggregator {

    /**
     * Aggregates the given analysis results into a single project-wide report.
     */
    fun aggregate(results: List<AnalysisResult>, config: CddConfig): AggregatedAnalysis {
        val allClassAnalyses = results.flatMap { it.classes }
        val allMethodAnalyses = allClassAnalyses.flatMap { it.methods }

        val totalFiles = results.size
        val totalClasses = allClassAnalyses.size
        val totalIcp = allClassAnalyses.sumOf { it.totalIcp }
        val averageIcp = if (totalClasses > 0) totalIcp / totalClasses else 0.0

        val classesOverLimit = allClassAnalyses.filter { it.totalIcp > config.limit }
        val largestClasses = allClassAnalyses.sortedByDescending { it.totalIcp }.take(10)

        val icpDistribution = IcpType.entries.associateWith { type ->
            allClassAnalyses.sumOf { it.icpBreakdown[type]?.size ?: 0 }
        }

        val slocStatistics = computeSlocStatistics(allClassAnalyses, allMethodAnalyses)
        val correlation = computeIcpSlocCorrelation(allClassAnalyses)
        val methodsOverSlocLimit = allMethodAnalyses.filter { it.isOverSlocLimit }

        val suggestions = generateSuggestions(allClassAnalyses, allMethodAnalyses, icpDistribution, correlation, config)

        return AggregatedAnalysis(
            totalFiles = totalFiles,
            totalClasses = totalClasses,
            totalIcp = totalIcp,
            averageIcp = averageIcp,
            classesOverLimit = classesOverLimit,
            icpDistribution = icpDistribution.mapKeys { it.key },
            largestClasses = largestClasses,
            slocMetrics = slocStatistics,
            icpSlocCorrelation = correlation,
            methodsOverSlocLimit = methodsOverSlocLimit,
            suggestions = suggestions
        )
    }

    private fun computeSlocStatistics(
        classes: List<ClassAnalysis>,
        methods: List<MethodAnalysis>
    ): SlocStatistics {
        if (classes.isEmpty()) {
            return SlocStatistics(0, 0.0, 0.0, 0, 0.0, emptyMap())
        }

        val totalSloc = classes.sumOf { it.sloc.total }
        val avgSlocPerClass = totalSloc.toDouble() / classes.size
        val avgSlocPerMethod =
            if (methods.isNotEmpty()) methods.sumOf { it.sloc.codeOnly }.toDouble() / methods.size else 0.0

        val methodSlocs = methods.map { it.sloc.total }.sorted()
        val medianSlocPerMethod = if (methodSlocs.isNotEmpty()) {
            methodSlocs[methodSlocs.size / 2]
        } else 0

        val slocVariance = classes.sumOf {
            val diff = it.sloc.total - avgSlocPerClass
            diff * diff
        } / classes.size
        val slocStdDev = sqrt(slocVariance)

        val distribution = calculateSlocDistribution(classes)

        return SlocStatistics(
            totalSloc = totalSloc,
            averageSlocPerClass = avgSlocPerClass,
            averageSlocPerMethod = avgSlocPerMethod,
            medianSlocPerMethod = medianSlocPerMethod,
            slocStdDev = slocStdDev,
            slocDistribution = distribution
        )
    }

    private fun calculateSlocDistribution(classes: List<ClassAnalysis>): Map<Int, Int> {
        val bucketSize = 50
        return classes.groupBy { (it.sloc.total / bucketSize) * bucketSize }
            .mapValues { it.value.size }
            .toSortedMap()
    }

    private fun computeIcpSlocCorrelation(classes: List<ClassAnalysis>): Double {
        if (classes.size < 2) return 0.0

        val avgIcp = classes.map { it.totalIcp }.average()
        val avgSloc = classes.map { it.sloc.total.toDouble() }.average()

        var numerator = 0.0
        var icpSquareSum = 0.0
        var slocSquareSum = 0.0

        for (cls in classes) {
            val icpDiff = cls.totalIcp - avgIcp
            val slocDiff = cls.sloc.total.toDouble() - avgSloc
            numerator += icpDiff * slocDiff
            icpSquareSum += icpDiff * icpDiff
            slocSquareSum += slocDiff * slocDiff
        }

        val denominator = sqrt(icpSquareSum * slocSquareSum)
        return if (denominator != 0.0) numerator / denominator else 0.0
    }

    private fun generateSuggestions(
        classes: List<ClassAnalysis>,
        methods: List<MethodAnalysis>,
        distribution: Map<IcpType, Int>,
        correlation: Double,
        config: CddConfig
    ): List<String> {
        val suggestions = mutableListOf<String>()
        val totalIcp = classes.sumOf { it.totalIcp }

        if (classes.isEmpty()) return emptyList()

        // 1. Overall Status
        val classesOverLimit = classes.filter { it.totalIcp > config.limit }
        if (classesOverLimit.isNotEmpty()) {
            suggestions.add("Refactor the ${classesOverLimit.size} classes that exceed the ICP limit of ${config.limit}.")
            val worstClass = classesOverLimit.maxByOrNull { it.totalIcp }
            if (worstClass != null) {
                suggestions.add(
                    "Prioritize '${worstClass.name}' as it has the highest complexity (${
                        String.format(
                            "%.1f",
                            worstClass.totalIcp
                        )
                    } ICP)."
                )
            }
        } else {
            suggestions.add("No classes exceed the complexity limit. Good job!")
        }

        // 2. ICP Type Analysis
        if (totalIcp > 0) {
            val exceptionIcp = distribution[IcpType.EXCEPTION_HANDLING] ?: 0
            if (exceptionIcp.toDouble() / totalIcp > 0.2) {
                suggestions.add("Exception handling accounts for >20% of total complexity. Consider a more centralized error handling strategy or using functional error handling.")
            }

            val couplingIcp =
                (distribution[IcpType.INTERNAL_COUPLING] ?: 0) + (distribution[IcpType.EXTERNAL_COUPLING] ?: 0) * 0.5
            if (couplingIcp / totalIcp > 0.4) {
                suggestions.add("Coupling accounts for a large portion of complexity. Consider extracting high-coupling logic into specialized services or using interfaces to decouple components.")
            }
        }

        // 3. Correlation Analysis
        if (correlation > 0.8 && classes.size >= 5) {
            suggestions.add("Strong correlation ($correlation) between SLOC and ICP detected. Making methods smaller (extracting methods) will likely reduce cognitive complexity directly.")
        } else if (correlation < 0.3 && classes.size >= 5 && classes.any { it.totalIcp > config.limit }) {
            suggestions.add("Low correlation ($correlation) between SLOC and ICP. Some classes are complex despite being small. Watch out for 'brain methods' with high density of logic.")
        }

        // 4. Method limits
        val overSloc = methods.filter { it.isOverSlocLimit }
        if (overSloc.isNotEmpty()) {
            suggestions.add("${overSloc.size} methods exceed the ${config.sloc.methodLimit} SLOC threshold. Breaking these down is highly recommended.")
            overSloc.take(3).forEach { method ->
                suggestions.add("  - Consider extracting logic from '${method.name}' (${method.sloc.total} SLOC).")
            }
        }

        val methodHighIcpThreshold = config.limit / 3.0
        val highIcpMethods = methods.filter { it.totalIcp > methodHighIcpThreshold }.sortedByDescending { it.totalIcp }
        if (highIcpMethods.isNotEmpty()) {
            highIcpMethods.take(3).forEach { method ->
                if (method.totalIcp > config.limit * 0.8) {
                    suggestions.add(
                        "Method '${method.name}' is highly complex (${
                            String.format(
                                "%.1f",
                                method.totalIcp
                            )
                        } ICP). Split it!"
                    )
                } else {
                    suggestions.add(
                        "Method '${method.name}' is approaching the complexity limit (${
                            String.format(
                                "%.1f",
                                method.totalIcp
                            )
                        } ICP)."
                    )
                }
            }
        }

        return suggestions
    }
}
