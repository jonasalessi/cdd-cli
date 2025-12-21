package com.cdd.analyzer.java

import com.cdd.core.config.CddConfig
import com.cdd.core.config.InternalCouplingConfig
import com.cdd.domain.IcpType
import io.kotest.core.spec.style.FunSpec
import io.kotest.matchers.shouldBe
import io.kotest.matchers.shouldNotBe
import java.io.File

class JavaAnalyzerTest : FunSpec({
    val analyzer = JavaAnalyzer()
    val config = CddConfig(
        limit = 10.0,
        icpTypes = IcpType.entries.associateWith { it.defaultWeight },
        internalCoupling = InternalCouplingConfig(packages = listOf("com.example.domain"))
    )

    test("should detect branches and conditions in SampleBranches.java") {
        val file = File("src/test/resources/java-samples/SampleBranches.java")
        val result = analyzer.analyze(file, config)
        
        result.totalIcp shouldBe 14.0 // 5 (if/else if/else) + 3 (switch) + 2 (ternary) + 4 (loops)
        
        val classAnalysis = result.classes.first()
        classAnalysis.name shouldBe "SampleBranches"
        
        // testIf: 3 branches (if, else if, else), 2 conditions (x>0, x<0) = 5 ICP
        val testIf = classAnalysis.methods.find { it.name == "testIf" }!!
        testIf.totalIcp shouldBe 5.0

        // testSwitch: 1 branch (switch) + 2 branches (cases) = 3 ICP
        val testSwitch = classAnalysis.methods.find { it.name == "testSwitch" }!!
        testSwitch.totalIcp shouldBe 3.0

        // testTernary: 1 branch, 1 condition = 2 ICP
        val testTernary = classAnalysis.methods.find { it.name == "testTernary" }!!
        testTernary.totalIcp shouldBe 2.0

        // testLoops: for (1 branch, 1 cond), while (1 branch, 1 cond) = 4
        val testLoops = classAnalysis.methods.find { it.name == "testLoops" }!!
        testLoops.totalIcp shouldBe 4.0
    }

    test("should detect complex conditions in SampleConditions.java") {
        val file = File("src/test/resources/java-samples/SampleConditions.java")
        val result = analyzer.analyze(file, config)
        
        val method = result.classes.first().methods.first()
        // Method contains two if statements, each with 1 branch and 3 conditions -> 8 ICP
        method.totalIcp shouldBe 8.0 
    }

    test("should detect exceptions in SampleExceptions.java") {
        val file = File("src/test/resources/java-samples/SampleExceptions.java")
        val result = analyzer.analyze(file, config)
        
        val classAnalysis = result.classes.find { it.name == "SampleExceptions" }
        classAnalysis shouldNotBe null
        
        val method = classAnalysis!!.methods.find { it.name == "testExceptions" }!!
        // try-catch-catch-finally -> 4 ICP
        method.totalIcp shouldBe 4.0
    }

    test("should detect coupling in SampleCoupling.java") {
        val file = File("src/test/resources/java-samples/SampleCoupling.java")
        val result = analyzer.analyze(file, config)
        
        val classAnalysis = result.classes.first()
        
        val internal = classAnalysis.icpBreakdown[IcpType.INTERNAL_COUPLING] ?: emptyList()
        val external = classAnalysis.icpBreakdown[IcpType.EXTERNAL_COUPLING] ?: emptyList()
        
        // Internal: User field, User parameter = 2.
        internal.size shouldBe 2 
        
        // External: List, String, ArrayList are all JDK types and should be excluded.
        external.size shouldBe 0
    }

    test("should calculate SLOC metrics correctly") {
        val file = File("src/test/resources/java-samples/SampleBranches.java")
        val result = analyzer.analyze(file, config)
        val classAnalysis = result.classes.first()
        
        classAnalysis.sloc.total shouldBe 37
        classAnalysis.sloc.codeOnly shouldBe 34
        classAnalysis.sloc.blankLines shouldBe 3
    }
})
