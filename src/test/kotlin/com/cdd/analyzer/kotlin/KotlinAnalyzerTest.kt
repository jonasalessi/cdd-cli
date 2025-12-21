package com.cdd.analyzer.kotlin

import com.cdd.core.config.*
import com.cdd.domain.IcpType
import io.kotest.core.spec.style.StringSpec
import io.kotest.matchers.shouldBe
import io.kotest.matchers.collections.shouldHaveSize
import io.kotest.matchers.ints.shouldBeGreaterThan
import java.io.File

class KotlinAnalyzerTest : StringSpec({
    val analyzer = KotlinAnalyzer()
    val config = CddConfig(
        limit = 10.0,
        icpTypes = IcpType.values().associateWith { it.defaultWeight },
        classTypeLimits = emptyMap(),
        internalCoupling = InternalCouplingConfig(autoDetect = true, packages = listOf("com.example")),
        externalCoupling = ExternalCouplingConfig(),
        include = listOf("**/*.kt"),
        exclude = emptyList(),
        reporting = ReportingConfig()
    )

    "should detect Kotlin-specific constructs in SampleConstructs.kt" {
        val file = File("src/test/resources/kotlin-samples/SampleConstructs.kt")
        val result = analyzer.analyze(file, config)
        
        result.classes shouldHaveSize 1
        val classAnalysis = result.classes.first()
        classAnalysis.name shouldBe "SampleConstructs"
        
        // when: 1 branch + 1 else branch = 2 codebase branches
        // elvis: 1 branch
        // safe call: 1 branch
        // if: 1 branch
        // Total branches = 2 (when) + 1 (elvis) + 1 (safe call) + 1 (if) = 5
        
        // conditions: 
        // when subject: 1 condition
        // if: 1 whole + 2 logical = 3
        // elvis: 1 condition
        // safe call: 1 condition
        // Total = 1 + 3 + 1 + 1 = 6
        
        classAnalysis.icpBreakdown[IcpType.CODE_BRANCH]?.size shouldBe 5
        classAnalysis.icpBreakdown[IcpType.CONDITION]?.size shouldBe 6
    }

    "should detect control flow in SampleControlFlow.kt" {
        val file = File("src/test/resources/kotlin-samples/SampleControlFlow.kt")
        val result = analyzer.analyze(file, config)
        
        val classAnalysis = result.classes.first()
        
        // for loop: 1 branch + 1 condition (loopRange)
        // while loop: 1 branch + 1 condition
        // if/else if/else: 3 branches + 2 conditions (2 if-wholes)
        // Total branches: 1 + 1 + 3 = 5
        // Total conditions: 1 + 1 + 2 = 4
        
        classAnalysis.icpBreakdown[IcpType.CODE_BRANCH]?.size shouldBe 5
        classAnalysis.icpBreakdown[IcpType.CONDITION]?.size shouldBe 4
    }

    "should detect exceptions in SampleExceptions.kt" {
        val file = File("src/test/resources/kotlin-samples/SampleExceptions.kt")
        val result = analyzer.analyze(file, config)
        
        val classAnalysis = result.classes.first()
        
        // try, catch, finally = 3 exception handling points
        classAnalysis.icpBreakdown[IcpType.EXCEPTION_HANDLING]?.size shouldBe 3
    }

    "should detect coupling in SampleCoupling.kt" {
        val file = File("src/test/resources/kotlin-samples/SampleCoupling.kt")
        val result = analyzer.analyze(file, config)
        
        val classAnalysis = result.classes.first()
        
        // Internal coupling: User
        // External coupling: Lib
        
        classAnalysis.icpBreakdown[IcpType.INTERNAL_COUPLING]?.size shouldBe 1
        classAnalysis.icpBreakdown[IcpType.EXTERNAL_COUPLING]?.size shouldBe 1
    }

    "should calculate SLOC metrics correctly" {
        val file = File("src/test/resources/kotlin-samples/SampleConstructs.kt")
        val result = analyzer.analyze(file, config)
        val classAnalysis = result.classes.first()
        
        classAnalysis.sloc.total shouldBeGreaterThan 0
        classAnalysis.sloc.codeOnly shouldBeGreaterThan 0
    }
})
