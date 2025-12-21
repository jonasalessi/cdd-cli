package com.cdd.e2e

import io.kotest.core.spec.style.FunSpec
import io.kotest.matchers.shouldBe
import io.kotest.matchers.string.shouldContain
import java.io.File
import java.util.concurrent.TimeUnit

class EndToEndTest : FunSpec({

    val projectRoot = File(".").absoluteFile
    val executable = if (System.getProperty("os.name").lowercase().contains("win")) {
        projectRoot.resolve("build/install/cdd-cli/bin/cdd-cli.bat")
    } else {
        projectRoot.resolve("build/install/cdd-cli/bin/cdd-cli")
    }

    fun runCli(vararg args: String, workingDir: File = projectRoot): CliResult {
        val process = ProcessBuilder(executable.absolutePath, *args)
            .directory(workingDir)
            .redirectErrorStream(true)
            .start()

        val output = process.inputStream.bufferedReader().readText()
        val finished = process.waitFor(15, TimeUnit.SECONDS)
        if (!finished) {
            process.destroyForcibly()
            throw RuntimeException("CLI timed out after 15 seconds. Output so far:\n$output")
        }

        if (process.exitValue() != 0 && !args.contains("--fail-on-violations")) {
            println("CLI Execution failed unexpectedly with code ${process.exitValue()}:\n$output")
        }

        return CliResult(process.exitValue(), output)
    }

    test("should analyze Kotlin sample and return success") {
        val sampleDir = projectRoot.resolve("src/test/resources/sample-kotlin")
        val result = runCli(sampleDir.absolutePath)
        
        result.exitCode shouldBe 0
        result.output shouldContain "Files analyzed:          1"
        result.output shouldContain "SimpleKotlin"
        result.output shouldContain "Total ICP:               3.0"
    }

    test("should fail on violations when --fail-on-violations is set") {
        val sampleDir = projectRoot.resolve("src/test/resources/sample-kotlin")
        // .cdd.yaml in that dir has limit: 2.0
        val result = runCli(sampleDir.absolutePath, "--fail-on-violations")
        
        result.exitCode shouldBe 1
        result.output shouldContain "Analysis failed: 1 violations found."
    }

    test("should analyze Java sample") {
        val sampleDir = projectRoot.resolve("src/test/resources/sample-java")
        val result = runCli(sampleDir.absolutePath)
        
        result.exitCode shouldBe 0
        result.output shouldContain "SimpleJava"
        result.output shouldContain "Total ICP:               3.0"
    }

    test("should analyze mixed project") {
        val sampleDir = projectRoot.resolve("src/test/resources/sample-mixed")
        val result = runCli(sampleDir.absolutePath)
        
        result.exitCode shouldBe 0
        result.output shouldContain "Files analyzed:          2"
        result.output shouldContain "Classes analyzed:        2"
        result.output shouldContain "Total ICP:               6.0"
    }

    test("should respect --format json and --output") {
        val sampleDir = projectRoot.resolve("src/test/resources/sample-kotlin")
        val outputFile = projectRoot.resolve("build/test-results/report.json")
        outputFile.parentFile.mkdirs()
        
        val result = runCli(sampleDir.absolutePath, "--format", "json", "--output", outputFile.absolutePath)
        
        result.exitCode shouldBe 0
        outputFile.exists() shouldBe true
        val json = outputFile.readText()
        json shouldContain "\"totalFiles\": 1"
        json shouldContain "\"name\": \"SimpleKotlin\""
    }

    test("should respect --include and --exclude") {
        val sampleDir = projectRoot.resolve("src/test/resources/sample-mixed")
        
        // Only include Kotlin
        val resultInclude = runCli(sampleDir.absolutePath, "--include", "*.kt")
        resultInclude.output shouldContain "Files analyzed:          1"
        resultInclude.output shouldContain "SimpleKotlin"

        // Exclude Kotlin
        val resultExclude = runCli(sampleDir.absolutePath, "--exclude", "*.kt")
        resultExclude.output shouldContain "Files analyzed:          1"
        resultExclude.output shouldContain "SimpleJava"
    }
})

data class CliResult(val exitCode: Int, val output: String)

private infix fun String.shouldNotContain(substring: String) {
    if (this.contains(substring)) {
        throw AssertionError("String contained \"$substring\" but it shouldn't have.\nOutput: $this")
    }
}
