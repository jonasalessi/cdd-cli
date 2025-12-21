package com.cdd.domain

import kotlinx.serialization.Serializable

/**
 * Project-wide aggregated analysis results.
 */
@Serializable
data class AggregatedAnalysis(
    val totalFiles: Int,
    val totalClasses: Int,
    val totalIcp: Double,
    val averageIcp: Double,
    val classesOverLimit: List<ClassAnalysis>,
    val icpDistribution: Map<IcpType, Int>,
    val largestClasses: List<ClassAnalysis>,
    val slocMetrics: SlocStatistics,
    val icpSlocCorrelation: Double,
    val methodsOverSlocLimit: List<MethodAnalysis>,
    val suggestions: List<String> = emptyList()
)

/**
 * Detailed SLOC statistics for the project.
 */
@Serializable
data class SlocStatistics(
    val totalSloc: Int,
    val averageSlocPerClass: Double,
    val averageSlocPerMethod: Double,
    val medianSlocPerMethod: Int,
    val slocStdDev: Double,
    val slocDistribution: Map<Int, Int> // Bucket start -> count
)
