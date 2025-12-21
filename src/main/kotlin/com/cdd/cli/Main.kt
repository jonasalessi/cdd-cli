package com.cdd.cli

import com.cdd.analyzer.java.JavaAnalyzer
import com.cdd.analyzer.kotlin.KotlinAnalyzer
import com.cdd.core.aggregator.IcpAggregator
import com.cdd.core.config.CddConfig
import com.cdd.core.config.ConfigurationManager
import com.cdd.core.config.OutputFormat
import com.cdd.core.registry.AnalyzerRegistry
import com.cdd.core.scanner.FileScanner
import com.cdd.core.scanner.PackageDetector
import com.cdd.domain.AnalysisResult
import com.github.ajalt.clikt.core.CliktCommand
import com.github.ajalt.clikt.parameters.arguments.argument
import com.github.ajalt.clikt.parameters.options.*
import com.github.ajalt.clikt.parameters.types.enum
import com.github.ajalt.clikt.parameters.types.file
import com.github.ajalt.clikt.parameters.types.int
import java.io.File
import kotlin.system.exitProcess

class CddCli : CliktCommand(name = "cdd-cli", help = "Cognitive-Driven Development Analyzer") {
    val path by argument(help = "Directory or file to analyze").file(mustExist = true, canBeFile = true, canBeDir = true)
    
    val limit by option(help = "ICP limit (default: 10)").int()
    val slocLimit by option("--sloc-limit", help = "SLOC limit for methods (default: 24)").int()
    val slocOnly by option("--sloc-only", help = "Show only SLOC analysis (no ICP)").flag(default = false)
    val format by option("--format", help = "Output format: console|json|xml|markdown (default: console)").enum<OutputFormat> { it.name.lowercase() }.default(OutputFormat.CONSOLE)
    val output by option("--output", help = "Output file (default: stdout)").file()
    val configPath by option("--config", help = "Config file path (default: .cdd.yml)").file(mustExist = true)
    val include by option("--include", help = "Include file pattern (can be repeated)").multiple()
    val exclude by option("--exclude", help = "Exclude file pattern (can be repeated)").multiple()
    val methodLevel by option("--method-level", help = "Show method-level analysis").flag(default = false)
    val failOnViolations by option("--fail-on-violations", help = "Exit with code 1 if violations found").flag(default = false)
    val verbose by option("-v", "--verbose", help = "Verbose output").flag(default = false)
    
    init {
        versionOption("0.1.0")
        registerAnalyzers()
        registerReporters()
    }

    private fun registerAnalyzers() {
        AnalyzerRegistry.register(JavaAnalyzer())
        AnalyzerRegistry.register(KotlinAnalyzer())
    }

    private fun registerReporters() {
        com.cdd.reporter.ReporterRegistry.register(com.cdd.reporter.ConsoleReporter())
        com.cdd.reporter.ReporterRegistry.register(com.cdd.reporter.JsonReporter())
        com.cdd.reporter.ReporterRegistry.register(com.cdd.reporter.XmlReporter())
        com.cdd.reporter.ReporterRegistry.register(com.cdd.reporter.MarkdownReporter())
    }

    override fun run() {
        if (path.isFile || path.isDirectory) {
            val baseDir = if (path.isDirectory) path else path.parentFile ?: File(".")
            val config = loadConfiguration(baseDir)
            
            if (verbose) {
                echo("Using configuration: $config")
            }

            val files = discoverFiles(config)
            if (files.isEmpty()) {
                echo("No files found to analyze.")
                return
            }

            if (verbose) echo("Analyzing ${files.size} files...")
            val results = analyzeFiles(files, config)
            val aggregatedResults = aggregateResults(results, config)

            generateAndOutputReport(aggregatedResults, config)
            handleExitCode(aggregatedResults)
        }
    }

    private fun generateAndOutputReport(results: com.cdd.domain.AggregatedAnalysis, config: CddConfig) {
        val reporter = com.cdd.reporter.ReporterRegistry.getReporter(config.reporting.format)
        val report = reporter.generate(results, config)
        
        val outputFile = config.reporting.outputFile
        if (outputFile != null) {
            File(outputFile).writeText(report)
            if (verbose) echo("Report saved to $outputFile")
        } else {
            echo(report)
        }
    }

    private fun loadConfiguration(baseDir: File): CddConfig {
        val baseConfig = if (configPath != null) {
            ConfigurationManager.loadConfigFile(configPath!!)
        } else {
            ConfigurationManager.loadConfig(baseDir)
        }

        val mergedConfig = baseConfig.copy(
            limit = limit ?: baseConfig.limit,
            sloc = baseConfig.sloc.copy(
                methodLimit = slocLimit ?: baseConfig.sloc.methodLimit
            ),
            include = if (include.isNotEmpty()) include else baseConfig.include,
            exclude = if (exclude.isNotEmpty()) exclude else baseConfig.exclude,
            reporting = baseConfig.reporting.copy(
                format = format,
                outputFile = output?.absolutePath ?: baseConfig.reporting.outputFile,
                verbose = verbose || baseConfig.reporting.verbose
            )
        )

        // Auto-detect internal packages if configured
        return if (mergedConfig.internalCoupling.autoDetect) {
            val files = discoverFiles(mergedConfig)
            val detectedPackages = PackageDetector.detectPackages(files)
            mergedConfig.copy(
                internalCoupling = mergedConfig.internalCoupling.copy(
                    packages = (mergedConfig.internalCoupling.packages + detectedPackages).distinct()
                )
            )
        } else {
            mergedConfig
        }
    }

    private fun discoverFiles(config: CddConfig): List<File> {
        return if (path.isFile) {
            if (AnalyzerRegistry.getAnalyzerFor(path) != null) listOf(path) else emptyList()
        } else {
            FileScanner(config.include, config.exclude).scan(path)
        }
    }

    private fun analyzeFiles(files: List<File>, config: CddConfig): List<AnalysisResult> {
        return files.mapNotNull { file ->
            val analyzer = AnalyzerRegistry.getAnalyzerFor(file)
            if (analyzer != null) {
                if (verbose) echo("Analyzing ${file.name}...")
                analyzer.analyze(file, config)
            } else {
                null
            }
        }
    }

    private fun aggregateResults(results: List<AnalysisResult>, config: CddConfig): com.cdd.domain.AggregatedAnalysis {
        return IcpAggregator().aggregate(results, config)
    }

    private fun handleExitCode(results: com.cdd.domain.AggregatedAnalysis) {
        if (failOnViolations && results.classesOverLimit.isNotEmpty()) {
            echo("\nAnalysis failed: ${results.classesOverLimit.size} violations found.")
            exitProcess(1)
        }
    }

}

fun main(args: Array<String>) = CddCli().main(args)
