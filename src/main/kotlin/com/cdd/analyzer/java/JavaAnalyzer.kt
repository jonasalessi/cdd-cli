package com.cdd.analyzer.java

import com.cdd.analyzer.LanguageAnalyzer
import com.cdd.core.config.CddConfig
import com.cdd.domain.*
import org.slf4j.LoggerFactory
import spoon.Launcher
import spoon.reflect.declaration.CtClass
import spoon.reflect.declaration.CtElement
import java.io.File

class JavaAnalyzer : LanguageAnalyzer {
    private val logger = LoggerFactory.getLogger(this::class.java)

    override val supportedExtensions: List<String> = listOf("java")
    override val languageName: String = "Java"

    override fun analyze(file: File, config: CddConfig): AnalysisResult {
        return try {
            val launcher = createLauncher(file)
            launcher.buildModel()
            val model = launcher.model

            val classes = mutableListOf<ClassAnalysis>()

            model.allTypes.filterIsInstance<CtClass<*>>().forEach { ctClass ->
                classes.add(analyzeClass(ctClass, config))
            }

            AnalysisResult(
                file = file.absolutePath,
                classes = classes,
                totalIcp = classes.sumOf { it.totalIcp }
            )
        } catch (e: Exception) {
            logger.error("Error analyzing ${file.name}: ${e.message}", e)
            AnalysisResult(
                file = file.absolutePath,
                classes = emptyList(),
                totalIcp = 0.0,
                errors = listOf(AnalysisError(file.absolutePath, null, e.message ?: "Unknown error", ErrorSeverity.ERROR))
            )
        }
    }

    private fun createLauncher(file: File): Launcher {
        val launcher = Launcher()
        launcher.environment.complianceLevel = 21
        launcher.environment.noClasspath = true
        launcher.environment.setCommentEnabled(true)
        launcher.addInputResource(file.absolutePath)
        return launcher
    }

    private fun analyzeClass(ctClass: CtClass<*>, config: CddConfig): ClassAnalysis {
        val scanner = JavaCtScanner(config)
        ctClass.accept(scanner)

        val classIcpInstances = scanner.icpInstances.values.flatten()

        val methods = ctClass.methods.map { ctMethod ->
            val methodRange = ctMethod.position.line..ctMethod.position.endLine
            val methodIcpInstances = classIcpInstances.filter { it.line in methodRange }

            val methodIcpBreakdown = methodIcpInstances.groupBy { it.type }

            MethodAnalysis(
                name = ctMethod.simpleName,
                lineRange = methodRange.toSerializable(),
                totalIcp = methodIcpInstances.sumOf { it.weight },
                icpBreakdown = methodIcpBreakdown,
                sloc = calculateSlocOf(ctMethod),
                isOverSlocLimit = false // Will be set by aggregator if needed
            )
        }

        val classIcpBreakdown = classIcpInstances.groupBy { it.type }
        val lineRange = ctClass.position.line..ctClass.position.endLine

        val totalIcp = classIcpInstances.sumOf { it.weight }

        return ClassAnalysis(
            name = ctClass.simpleName,
            packageName = ctClass.`package`?.qualifiedName ?: "",
            lineRange = lineRange.toSerializable(),
            totalIcp = totalIcp,
            icpBreakdown = classIcpBreakdown,
            methods = methods,
            isOverLimit = totalIcp > config.limit,
            sloc = calculateSlocOf(ctClass)
        )
    }

    private fun calculateSlocOf(ctElement: CtElement): SlocMetrics {
        return try {
            val position = ctElement.position
            if (position.isValidPosition) {
                val content = position.compilationUnit?.originalSourceCode ?: ""
                calculateSloc(content, position.line, position.endLine)
            } else {
                SlocMetrics(0, 0, 0, 0, 0)
            }
        } catch (e: Exception) {
            SlocMetrics(0, 0, 0, 0, 0)
        }
    }

    private fun calculateSloc(fullContent: String, startLine: Int, endLine: Int): SlocMetrics {
        val lines = fullContent.lines().subList(startLine - 1, endLine)

        var total = 0
        var codeOnly = 0
        var comments = 0
        var blankLines = 0

        var inBlockComment = false

        lines.forEach { line ->
            total++
            val trimmed = line.trim()

            if (trimmed.isEmpty()) {
                blankLines++
            } else if (trimmed.startsWith("//")) {
                comments++
            } else if (trimmed.startsWith("/*")) {
                comments++
                if (!trimmed.endsWith("*/")) inBlockComment = true
            } else if (inBlockComment) {
                comments++
                if (trimmed.endsWith("*/")) inBlockComment = false
            } else {
                codeOnly++
            }
        }

        return SlocMetrics(
            total = total,
            codeOnly = codeOnly,
            withComments = codeOnly + comments,
            comments = comments,
            blankLines = blankLines
        )
    }
}
