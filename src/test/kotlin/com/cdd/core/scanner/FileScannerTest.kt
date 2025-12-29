package com.cdd.core.scanner

import com.cdd.analyzer.LanguageAnalyzer
import com.cdd.core.config.CddConfig
import com.cdd.core.registry.AnalyzerRegistry
import com.cdd.domain.AnalysisResult
import io.kotest.core.spec.style.DescribeSpec
import io.kotest.matchers.collections.shouldContainExactlyInAnyOrder
import io.kotest.matchers.shouldBe
import io.mockk.mockk
import java.io.File
import java.nio.file.Files

class FileScannerTest : DescribeSpec({
    val tempDir = Files.createTempDirectory("scanner-test").toFile()

    beforeSpec {
        AnalyzerRegistry.clear()
        AnalyzerRegistry.register(object : LanguageAnalyzer {
            override val supportedExtensions = listOf("java", "kt")
            override val languageName = "Dummy"
            override fun analyze(file: File, config: CddConfig): AnalysisResult = mockk()
            override fun stripComments(line: String): String = line
        })
    }

    afterSpec {
        tempDir.deleteRecursively()
        AnalyzerRegistry.clear()
    }

    describe("FileScanner") {
        it("should scan files recursively and filter by extension") {
            val srcDir = File(tempDir, "src").apply { mkdirs() }
            val file1 = File(srcDir, "Test1.java").apply { writeText("package com.test;\nclass Test1 {}") }
            val file2 = File(srcDir, "Test2.kt").apply { writeText("package com.test;\nclass Test2") }
            val file3 = File(srcDir, "Test3.txt").apply { writeText("Not a source file") }

            val scanner = FileScanner(emptyList(), emptyList())
            val results = scanner.scan(tempDir)

            results.size shouldBe 2
            results.map { it.name } shouldContainExactlyInAnyOrder listOf("Test1.java", "Test2.kt")
        }

        it("should respect include patterns with nested files") {
            val srcDir = File(tempDir, "include-test").apply { mkdirs() }
            val nestedDir = File(srcDir, "nested").apply { mkdirs() }
            File(nestedDir, "Match.java").apply { writeText("package com.match;\nclass Match {}") }
            File(srcDir, "Ignore.java").apply { writeText("package com.ignore;\nclass Ignore {}") }

            val scanner = FileScanner(listOf("**/Match.java"), emptyList())
            val results = scanner.scan(srcDir)

            results.size shouldBe 1
            results[0].name shouldBe "Match.java"
        }

        it("should respect exclude patterns with nested files") {
            val srcDir = File(tempDir, "exclude-test").apply { mkdirs() }
            val nestedDir = File(srcDir, "nested").apply { mkdirs() }
            File(nestedDir, "Keep.java").apply { writeText("package com.keep;\nclass Keep {}") }
            File(nestedDir, "Skip.java").apply { writeText("package com.skip;\nclass Skip {}") }

            val scanner = FileScanner(emptyList(), listOf("**/Skip.java"))
            val results = scanner.scan(srcDir)

            results.size shouldBe 1
            results[0].name shouldBe "Keep.java"
        }
    }

    describe("PackageDetector") {
        it("should detect unique packages from files") {
            val pkgDir = File(tempDir, "pkg-test").apply { mkdirs() }
            val f1 = File(pkgDir, "F1.java").apply { writeText("package com.example.foo;\nclass F1 {}") }
            val f2 = File(pkgDir, "F2.java").apply { writeText("package com.example.bar;\nclass F2 {}") }
            val f3 = File(pkgDir, "F3.java").apply { writeText("package com.example.foo;\nclass F3 {}") }

            val packages = PackageDetector.detectPackages(listOf(f1, f2, f3))
            
            packages shouldContainExactlyInAnyOrder listOf("com.example.foo", "com.example.bar")
        }
    }
})
