package com.cdd.analyzer.kotlin

import com.cdd.core.config.CddConfig
import com.cdd.core.config.InternalCouplingConfig
import com.cdd.core.config.ReportingConfig
import com.cdd.domain.IcpType
import io.kotest.core.spec.style.StringSpec
import io.kotest.matchers.collections.shouldHaveSize
import io.kotest.matchers.ints.shouldBeGreaterThan
import io.kotest.matchers.shouldBe
import java.io.File

class KotlinAnalyzerTest : StringSpec({
    val analyzer = KotlinAnalyzer()
    val config = CddConfig(
        metrics = mapOf("kotlin" to mapOf(".*" to mapOf("code_branch" to 1.0))),
        icpLimits = mapOf("kotlin" to mapOf(".*" to 10.0)),
        internalCoupling = InternalCouplingConfig(autoDetect = true, packages = listOf("com.example", "com.challenge")),
        reporting = ReportingConfig()
    )

    "should detect Kotlin-specific constructs in SampleConstructs.kt" {
        val file = File("src/test/resources/kotlin-samples/SampleConstructs.kt")
        val result = analyzer.analyze(file, config)

        result.classes shouldHaveSize 1
        val classAnalysis = result.classes.first()
        classAnalysis.name shouldBe "SampleConstructs"

        classAnalysis.icpBreakdown[IcpType.CODE_BRANCH]?.size shouldBe 5
        classAnalysis.icpBreakdown[IcpType.CONDITION]?.size shouldBe 6
    }

    "should detect control flow in SampleControlFlow.kt" {
        val file = File("src/test/resources/kotlin-samples/SampleControlFlow.kt")
        val result = analyzer.analyze(file, config)

        val classAnalysis = result.classes.first()

        classAnalysis.icpBreakdown[IcpType.CODE_BRANCH]?.size shouldBe 5
        classAnalysis.icpBreakdown[IcpType.CONDITION]?.size shouldBe 4
    }

    "should detect exceptions in SampleExceptions.kt" {
        val file = File("src/test/resources/kotlin-samples/SampleExceptions.kt")
        val result = analyzer.analyze(file, config)

        val classAnalysis = result.classes.first()

        classAnalysis.icpBreakdown[IcpType.EXCEPTION_HANDLING]?.size shouldBe 3
    }

    "should detect coupling in SampleCoupling.kt" {
        val file = File("src/test/resources/kotlin-samples/com/examples/SampleCoupling.kt")
        val result = analyzer.analyze(file, config)

        val classAnalysis = result.classes.first()

        // Internal coupling: User
        // Internal coupling: House
        // Internal coupling: InternalClass

        classAnalysis.icpBreakdown[IcpType.INTERNAL_COUPLING]?.size shouldBe 3
    }

    "should not count external annotations as internal coupling" {
        val file = File("src/test/resources/kotlin-samples/com/examples/SampleAnnotationCoupling.kt")
        val result = analyzer.analyze(file, config)

        val classAnalysis = result.classes.first()

        val internalCouplings = classAnalysis.icpBreakdown[IcpType.INTERNAL_COUPLING]?.sumOf { it.weight }

        // Expected internal couplings:
        // Only 1 for FieldValidationException in isNameOk() and isIdOk()
        internalCouplings shouldBe 1
    }

    "should calculate SLOC metrics correctly" {
        val file = File("src/test/resources/kotlin-samples/SampleConstructs.kt")
        val result = analyzer.analyze(file, config)
        val classAnalysis = result.classes.first()

        classAnalysis.sloc.total shouldBeGreaterThan 0
        classAnalysis.sloc.codeOnly shouldBeGreaterThan 0
    }
})
