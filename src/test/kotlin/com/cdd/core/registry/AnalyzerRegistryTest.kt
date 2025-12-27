package com.cdd.core.registry

import com.cdd.analyzer.LanguageAnalyzer
import com.cdd.core.config.CddConfig
import com.cdd.domain.AnalysisResult
import io.kotest.core.spec.style.DescribeSpec
import io.kotest.matchers.shouldBe
import io.kotest.matchers.shouldNotBe
import io.mockk.mockk
import java.io.File

class AnalyzerRegistryTest : DescribeSpec({
    
    beforeEach {
        AnalyzerRegistry.clear()
    }

    describe("AnalyzerRegistry") {
        it("should register and dispatch to the correct analyzer") {
            val javaAnalyzer = object : LanguageAnalyzer {
                override val supportedExtensions = listOf("java")
                override val languageName = "Java"
                override fun analyze(file: File, config: CddConfig): AnalysisResult = mockk()
            }
            
            AnalyzerRegistry.register(javaAnalyzer)
            
            val testFile = File("Test.java")
            val analyzer = AnalyzerRegistry.getAnalyzerFor(testFile)
            
            analyzer shouldNotBe null
            analyzer?.languageName shouldBe "Java"
        }

        it("should return null for unsupported file extensions") {
            val javaAnalyzer = object : LanguageAnalyzer {
                override val supportedExtensions = listOf("java")
                override val languageName = "Java"
                override fun analyze(file: File, config: CddConfig): AnalysisResult = mockk()
            }
            
            AnalyzerRegistry.register(javaAnalyzer)
            
            val testFile = File("Test.kt")
            val analyzer = AnalyzerRegistry.getAnalyzerFor(testFile)
            
            analyzer shouldBe null
        }
    }
})
