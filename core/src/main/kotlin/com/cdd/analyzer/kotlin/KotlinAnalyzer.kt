package com.cdd.analyzer.kotlin

import com.cdd.analyzer.AbstractLanguageAnalyzer
import com.cdd.analyzer.LanguageAnalyzer
import com.cdd.core.config.CddConfig
import com.cdd.core.util.CommentUtils
import com.cdd.domain.*
import org.jetbrains.kotlin.cli.common.CLIConfigurationKeys
import org.jetbrains.kotlin.cli.common.messages.MessageRenderer
import org.jetbrains.kotlin.cli.common.messages.PrintingMessageCollector
import org.jetbrains.kotlin.cli.jvm.compiler.EnvironmentConfigFiles
import org.jetbrains.kotlin.cli.jvm.compiler.KotlinCoreEnvironment
import org.jetbrains.kotlin.com.intellij.openapi.util.Disposer
import org.jetbrains.kotlin.config.CompilerConfiguration
import org.jetbrains.kotlin.psi.*
import org.jetbrains.kotlin.psi.psiUtil.startOffset
import java.io.File

class KotlinAnalyzer : AbstractLanguageAnalyzer() {
    override val supportedExtensions: List<String> = listOf("kt", "kts")
    override val languageName: String = "Kotlin"

    override fun analyze(file: File, config: CddConfig): AnalysisResult {
        return try {
            val disposable = Disposer.newDisposable()
            val configuration = CompilerConfiguration()
            configuration.put(
                CLIConfigurationKeys.MESSAGE_COLLECTOR_KEY,
                PrintingMessageCollector(System.err, MessageRenderer.PLAIN_RELATIVE_PATHS, false)
            )

            val environment = KotlinCoreEnvironment.createForProduction(
                disposable,
                configuration,
                EnvironmentConfigFiles.JVM_CONFIG_FILES
            )

            val content = file.readText()
            val ktFile = KtPsiFactory(environment.project).createFile(content)

            val classes = mutableListOf<ClassAnalysis>()
            ktFile.accept(object : KtTreeVisitorVoid() {
                override fun visitClass(klass: KtClass) {
                    super.visitClass(klass)
                    classes.add(analyzeClass(klass, content, file, config))
                }
            })

            Disposer.dispose(disposable)

            AnalysisResult(
                file = file.absolutePath,
                classes = classes,
                totalIcp = classes.sumOf { it.totalIcp }
            )
        } catch (e: Exception) {
            e.printStackTrace()
            AnalysisResult(
                file = file.absolutePath,
                classes = emptyList(),
                totalIcp = 0.0,
                errors = listOf(
                    AnalysisError(
                        file.absolutePath,
                        message = e.message ?: "Unknown error",
                        severity = ErrorSeverity.ERROR
                    )
                )
            )
        }
    }



    private fun analyzeClass(ktClass: KtClass, fullContent: String, file: File, config: CddConfig): ClassAnalysis {
        val ktFile = ktClass.containingFile as KtFile
        // Resolve limits
        val weights = resolveWeights(file, config)
        val scanner = KotlinIcpScanner(fullContent, config, ktFile, weights, ktClass.fqName?.asString())
        ktClass.accept(scanner)

        val classIcpInstances = scanner.getIcpInstances()
        val totalIcp = classIcpInstances.sumOf { it.weight }
        val classIcpBreakdown = classIcpInstances.groupBy { it.type }

        val startLine = getLineNumber(fullContent, ktClass.startOffset)
        val endLine = getLineNumber(fullContent, ktClass.textRange.endOffset)
        val lineRange = (startLine..endLine).toSerializable()

        val classLimit = resolveIcpLimit(file, config) ?: Double.MAX_VALUE
        val overLimit = totalIcp > classLimit

        val methods = mutableListOf<MethodAnalysis>()
        ktClass.accept(object : KtTreeVisitorVoid() {
            override fun visitNamedFunction(function: KtNamedFunction) {
                // Only process methods directly in this class
                if (function.parent == ktClass || function.parent is KtClassBody && function.parent.parent == ktClass) {
                    val methodStart = getLineNumber(fullContent, function.startOffset)
                    val methodEnd = getLineNumber(fullContent, function.textRange.endOffset)
                    val methodRange = methodStart..methodEnd

                    val methodIcpInstances = classIcpInstances.filter { it.line in methodRange }
                    val methodSloc = calculateSloc(fullContent, methodStart, methodEnd)
                    
                    val methodBreakdown = methodIcpInstances.groupBy { it.type }
                    
                    methods.add(
                        MethodAnalysis(
                            name = function.name ?: "Unknown",
                            className = ktClass.name ?: "Unknown",
                            lineRange = methodRange.toSerializable(),
                            totalIcp = methodIcpInstances.sumOf { it.weight },
                            icpBreakdown = methodBreakdown,
                            sloc = methodSloc,
                            isOverSlocLimit = methodSloc.codeOnly > config.sloc.methodLimit
                        )
                    )
                }
            }
        })

        return ClassAnalysis(
            name = ktClass.name ?: "Unknown",
            packageName = (ktClass.containingFile as? KtFile)?.packageFqName?.asString() ?: "",
            lineRange = lineRange,
            totalIcp = totalIcp,
            icpBreakdown = classIcpBreakdown,
            methods = methods,
            isOverLimit = overLimit,
            sloc = calculateSloc(fullContent, startLine, endLine)
        )
    }

    private fun getLineNumber(content: String, offset: Int): Int {
        if (offset < 0) return 1
        val safeOffset = offset.coerceAtMost(content.length)
        return content.substring(0, safeOffset).count { it == '\n' } + 1
    }

    private fun calculateSloc(fullContent: String, startLine: Int, endLine: Int): SlocMetrics {
        val allLines = fullContent.lines()
        val rangeStart = (startLine - 1).coerceAtLeast(0)
        val rangeEnd = (endLine - 1).coerceAtMost(allLines.size - 1)

        if (rangeStart > rangeEnd) return SlocMetrics(0, 0, 0, 0, 0)

        val targetLines = allLines.subList(rangeStart, rangeEnd + 1)

        var total = 0
        var codeOnly = 0
        var comments = 0
        var blankLines = 0

        var inBlockComment = false

        targetLines.forEach { line ->
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

    override fun stripComments(line: String): String {
        val stripFn = { l: String ->
            CommentUtils.stripLineComment(
                CommentUtils.stripBlockComments(l)
            )
        }
        return if (CommentUtils.hasCode(line, stripFn)) {
            stripFn(line).trimEnd()
        } else {
            line
        }
    }
}
