package com.cdd.analyzer.kotlin

import com.cdd.analyzer.LanguageAnalyzer
import com.cdd.core.config.CddConfig
import com.cdd.domain.*
import org.jetbrains.kotlin.cli.common.CLIConfigurationKeys
import org.jetbrains.kotlin.cli.common.messages.MessageRenderer
import org.jetbrains.kotlin.cli.common.messages.PrintingMessageCollector
import org.jetbrains.kotlin.cli.jvm.compiler.EnvironmentConfigFiles
import org.jetbrains.kotlin.cli.jvm.compiler.KotlinCoreEnvironment
import org.jetbrains.kotlin.com.intellij.openapi.util.Disposer
import org.jetbrains.kotlin.com.intellij.psi.PsiElement
import org.jetbrains.kotlin.config.CompilerConfiguration
import org.jetbrains.kotlin.lexer.KtTokens
import org.jetbrains.kotlin.psi.*
import org.jetbrains.kotlin.psi.psiUtil.startOffset
import java.io.File

class KotlinAnalyzer : LanguageAnalyzer {
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
                    classes.add(analyzeClass(klass, content, config))
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

    private fun analyzeClass(ktClass: KtClass, fullContent: String, config: CddConfig): ClassAnalysis {
        val scanner = IcpScanner(fullContent, config)
        val ktFile = ktClass.containingFile as KtFile
        scanner.setKtFile(ktFile)
        ktClass.accept(scanner)

        val classIcpInstances = scanner.getIcpInstances()
        val totalIcp = classIcpInstances.sumOf { it.weight }

        val startLine = getLineNumber(fullContent, ktClass.startOffset)
        val endLine = getLineNumber(fullContent, ktClass.textRange.endOffset)
        val lineRange = (startLine..endLine).toSerializable()

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

                    methods.add(
                        MethodAnalysis(
                            name = function.name ?: "Unknown",
                            lineRange = methodRange.toSerializable(),
                            totalIcp = methodIcpInstances.sumOf { it.weight },
                            icpBreakdown = methodIcpInstances.groupBy { it.type },
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
            icpBreakdown = classIcpInstances.groupBy { it.type },
            methods = methods,
            isOverLimit = totalIcp > config.limit,
            sloc = calculateSloc(fullContent, startLine, endLine)
        )
    }

    private fun getLineNumber(content: String, offset: Int): Int {
        if (offset < 0) return 1
        val safeOffset = offset.coerceAtMost(content.length)
        return content.substring(0, safeOffset).count { it == '\n' } + 1
    }

    private fun getColumnNumber(content: String, offset: Int): Int {
        if (offset <= 0) return 1
        val safeOffset = offset.coerceAtMost(content.length)
        val lastNewLine = content.substring(0, safeOffset).lastIndexOf('\n')
        return if (lastNewLine == -1) safeOffset + 1 else safeOffset - lastNewLine
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


    private inner class IcpScanner(val fullContent: String, val config: CddConfig) : KtTreeVisitorVoid() {
        private val icpInstances = mutableListOf<IcpInstance>()

        // it.containingFile is tricky inside IcpScanner initialization. 
        // I'll just pass the ktFile to the scanner.
        private var currentKtFile: KtFile? = null

        fun getIcpInstances() = icpInstances

        fun setKtFile(ktFile: KtFile) {
            currentKtFile = ktFile
        }

        private val imports: Map<String, String> by lazy {
            currentKtFile?.importDirectives?.mapNotNull { import ->
                val fqName = import.importedFqName?.asString()
                val simpleName = import.importedFqName?.shortName()?.asString()
                if (fqName != null && simpleName != null) simpleName to fqName else null
            }?.toMap() ?: emptyMap()
        }

        private fun addInstance(type: IcpType, element: PsiElement, description: String) {
            val line = getLineNumber(fullContent, element.startOffset)
            val column = getColumnNumber(fullContent, element.startOffset)
            val weight = config.icpTypes[type] ?: type.defaultWeight

            File("cdd_debug.log").appendText("CDD_DEBUG_KOTLIN: $type at line $line: $description\n")
            icpInstances.add(IcpInstance(type, line, column, description, weight))
        }

        override fun visitIfExpression(expression: KtIfExpression) {
            addInstance(IcpType.CODE_BRANCH, expression, "if branch")
            expression.condition?.let { analyzeCondition(it) }

            val elseExpr = expression.`else`
            if (elseExpr != null) {
                // If the else branch is a block containing only an if, it's an else-if chain
                val isElseIf =
                    elseExpr is KtIfExpression || (elseExpr is KtBlockExpression && elseExpr.statements.size == 1 && elseExpr.statements[0] is KtIfExpression)
                if (!isElseIf) {
                    addInstance(IcpType.CODE_BRANCH, elseExpr, "else branch")
                }
            }
            super.visitIfExpression(expression)
        }

        override fun visitWhenExpression(expression: KtWhenExpression) {
            addInstance(IcpType.CODE_BRANCH, expression, "when branch")
            expression.subjectExpression?.let { analyzeCondition(it) }

            expression.entries.forEach { entry ->
                if (entry.isElse) {
                    addInstance(IcpType.CODE_BRANCH, entry, "else branch")
                }
            }
            super.visitWhenExpression(expression)
        }

        override fun visitForExpression(expression: KtForExpression) {
            addInstance(IcpType.CODE_BRANCH, expression, "for loop")
            expression.loopRange?.let { analyzeCondition(it) }
            super.visitForExpression(expression)
        }

        override fun visitWhileExpression(expression: KtWhileExpression) {
            addInstance(IcpType.CODE_BRANCH, expression, "while loop")
            expression.condition?.let { analyzeCondition(it) }
            super.visitWhileExpression(expression)
        }

        override fun visitDoWhileExpression(expression: KtDoWhileExpression) {
            addInstance(IcpType.CODE_BRANCH, expression, "do-while loop")
            expression.condition?.let { analyzeCondition(it) }
            super.visitDoWhileExpression(expression)
        }

        override fun visitTryExpression(expression: KtTryExpression) {
            addInstance(IcpType.EXCEPTION_HANDLING, expression, "try block")
            expression.finallyBlock?.let {
                addInstance(IcpType.EXCEPTION_HANDLING, it, "finally block")
            }
            super.visitTryExpression(expression)
        }

        override fun visitCatchSection(catchClause: KtCatchClause) {
            addInstance(IcpType.EXCEPTION_HANDLING, catchClause, "catch block")
            super.visitCatchSection(catchClause)
        }

        override fun visitBinaryExpression(expression: KtBinaryExpression) {
            val operationToken = expression.operationToken
            if (operationToken == KtTokens.ELVIS) {
                addInstance(IcpType.CODE_BRANCH, expression, "elvis operator")
                addInstance(IcpType.CONDITION, expression, "elvis condition")
            } else if (operationToken == KtTokens.ANDAND || operationToken == KtTokens.OROR) {
                addInstance(IcpType.CONDITION, expression, "logical operator ${expression.operationReference.text}")
            }
            super.visitBinaryExpression(expression)
        }

        override fun visitSafeQualifiedExpression(expression: KtSafeQualifiedExpression) {
            addInstance(IcpType.CODE_BRANCH, expression, "safe call")
            addInstance(IcpType.CONDITION, expression, "safe call condition")
            super.visitSafeQualifiedExpression(expression)
        }

        override fun visitCallExpression(expression: KtCallExpression) {
            val callee = expression.calleeExpression
            if (callee is KtNameReferenceExpression) {
                val text = callee.getReferencedName()
                if (!isJdkType(text)) {
                    val resolvedFqName = imports[text] ?: text
                    if (isInternal(resolvedFqName)) {
                        addInstance(IcpType.INTERNAL_COUPLING, expression, "Internal coupling (call): $resolvedFqName")
                    } else if (resolvedFqName.contains('.') || !isCommonType(text)) {
                        addInstance(IcpType.EXTERNAL_COUPLING, expression, "External coupling (call): $resolvedFqName")
                    }
                }
            }
            super.visitCallExpression(expression)
        }

        override fun visitConstructorCalleeExpression(constructorCalleeExpression: KtConstructorCalleeExpression) {
            constructorCalleeExpression.typeReference?.let { analyzeTypeReference(it) }
            super.visitConstructorCalleeExpression(constructorCalleeExpression)
        }

        override fun visitTypeReference(typeReference: KtTypeReference) {
            analyzeTypeReference(typeReference)
            super.visitTypeReference(typeReference)
        }

        private fun analyzeTypeReference(typeReference: KtTypeReference) {
            val text = typeReference.text.substringBefore('<').substringBefore('?').trim()
            if (text.isEmpty() || isJdkType(text)) return

            val resolvedFqName = imports[text] ?: text
            if (isInternal(resolvedFqName)) {
                addInstance(IcpType.INTERNAL_COUPLING, typeReference, "Internal coupling: $resolvedFqName")
            } else if (resolvedFqName.contains('.') || !isCommonType(text)) {
                // If it has a dot, it was either imported or fully qualified -> definite external coupling
                // If it doesn't have a dot but it's not a common type, it might be external but we're less sure without BindingContext
                // However, for testing purposes, we'll count it if it's not a common type.
                addInstance(IcpType.EXTERNAL_COUPLING, typeReference, "External coupling: $resolvedFqName")
            }
        }

        private fun analyzeCondition(element: PsiElement) {
            // Only add as a condition if it's not already handled (like logical operators)
            // Actually, to match Java's high ICP counts, we'll keep adding it for now.
            addInstance(IcpType.CONDITION, element, "condition expression")
        }

        private fun isJdkType(qualifiedName: String): Boolean {
            if (qualifiedName.startsWith("java.") ||
                qualifiedName.startsWith("javax.") ||
                qualifiedName.startsWith("kotlin.")
            ) return true

            val commonTypes = listOf(
                "String", "Int", "Long", "Boolean", "Double", "Float", "Byte", "Short", "Char",
                "List", "Map", "Set", "Any", "Unit", "Array", "Exception", "RuntimeException",
                "Error", "ArithmeticException", "NullPointerException", "IllegalArgumentException",
                "IllegalStateException", "ArrayList", "HashMap", "HashSet",
                "println", "print", "require", "check", "error", "assert", "lazy", "run", "let", "with", "apply", "also"
            )
            return commonTypes.contains(qualifiedName)
        }

        private fun isCommonType(name: String): Boolean {
            return listOf("String", "Int", "Long", "Boolean", "Double", "Float", "Any", "Unit").contains(name)
        }

        private fun isInternal(qualifiedName: String): Boolean {
            return config.internalCoupling.packages.any { pkg ->
                qualifiedName.startsWith("$pkg.") || qualifiedName == pkg
            }
        }
    }

    override fun stripComments(line: String): String {
        val stripFn = { l: String ->
            com.cdd.core.util.CommentUtils.stripLineComment(
                com.cdd.core.util.CommentUtils.stripBlockComments(l)
            )
        }
        return if (com.cdd.core.util.CommentUtils.hasCode(line, stripFn)) {
            stripFn(line).trimEnd()
        } else {
            line
        }
    }
}
