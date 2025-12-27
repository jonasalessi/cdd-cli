package com.cdd.analyzer.java

import com.cdd.core.config.CddConfig
import com.cdd.domain.IcpInstance
import com.cdd.domain.IcpType
import spoon.reflect.code.*
import spoon.reflect.declaration.CtElement
import spoon.reflect.reference.CtPackageReference
import spoon.reflect.reference.CtTypeReference
import spoon.reflect.visitor.CtScanner

class JavaCtScanner(val config: CddConfig) : CtScanner() {
    val icpInstances = mutableMapOf<IcpType, MutableList<IcpInstance>>()

    private fun addInstance(type: IcpType, element: CtElement, description: String) {
        val position = element.position
        if (!position.isValidPosition) return

        val weight = config.icpTypes[type] ?: type.defaultWeight
        val instance = IcpInstance(
            type = type,
            line = position.line,
            column = position.column,
            description = description,
            weight = weight
        )
        icpInstances.getOrPut(type) { mutableListOf() }.add(instance)
    }

    override fun visitCtIf(ifElement: CtIf) {
        addInstance(IcpType.CODE_BRANCH, ifElement, "if branch")
        analyzeCondition(ifElement.condition)

        val elseBlock = ifElement.getElseStatement<CtStatement>()
        if (elseBlock != null) {
            // If the else block is an 'if' or a block containing only an 'if', it's an 'else if' chain.
            // We don't count it as a separate 'else branch' here because the next 'if' will count itself.
            val isElseIf =
                elseBlock is CtIf || (elseBlock is CtBlock<*> && elseBlock.statements.size == 1 && elseBlock.statements[0] is CtIf)
            if (!isElseIf) {
                addInstance(IcpType.CODE_BRANCH, elseBlock, "else branch")
            }
        }
        super.visitCtIf(ifElement)
    }

    override fun <T : Any?> visitCtSwitch(switchElement: CtSwitch<T>) {
        addInstance(IcpType.CODE_BRANCH, switchElement, "switch branch")
        super.visitCtSwitch(switchElement)
    }

    override fun <T : Any?> visitCtCase(caseElement: CtCase<T>) {
        if (caseElement.caseExpressions.isNotEmpty()) {
            addInstance(IcpType.CODE_BRANCH, caseElement, "case branch")
        }
        super.visitCtCase(caseElement)
    }

    override fun <T : Any?> visitCtConditional(conditional: CtConditional<T>) {
        addInstance(IcpType.CODE_BRANCH, conditional, "ternary operator")
        analyzeCondition(conditional.condition)
        super.visitCtConditional(conditional)
    }

    override fun visitCtFor(forLoop: CtFor) {
        addInstance(IcpType.CODE_BRANCH, forLoop, "for loop")
        analyzeCondition(forLoop.expression)
        super.visitCtFor(forLoop)
    }

    override fun visitCtForEach(foreach: CtForEach) {
        addInstance(IcpType.CODE_BRANCH, foreach, "foreach loop")
        super.visitCtForEach(foreach)
    }

    override fun visitCtWhile(whileLoop: CtWhile) {
        addInstance(IcpType.CODE_BRANCH, whileLoop, "while loop")
        analyzeCondition(whileLoop.loopingExpression)
        super.visitCtWhile(whileLoop)
    }

    override fun visitCtDo(doLoop: CtDo) {
        addInstance(IcpType.CODE_BRANCH, doLoop, "do-while loop")
        analyzeCondition(doLoop.loopingExpression)
        super.visitCtDo(doLoop)
    }

    override fun visitCtTry(tryBlock: CtTry) {
        addInstance(IcpType.EXCEPTION_HANDLING, tryBlock, "try block")
        if (tryBlock.finalizer != null) {
            addInstance(IcpType.EXCEPTION_HANDLING, tryBlock.finalizer, "finally block")
        }
        super.visitCtTry(tryBlock)
    }

    override fun visitCtCatch(catchBlock: CtCatch) {
        addInstance(IcpType.EXCEPTION_HANDLING, catchBlock, "catch block")
        super.visitCtCatch(catchBlock)
    }

    override fun <T : Any?> visitCtTypeReference(reference: CtTypeReference<T>) {
        if (isCouplingType(reference)) {
            val qualifiedName = reference.qualifiedName
            val isInternal = config.internalCoupling.packages.any { qualifiedName.startsWith(it) }

            if (isInternal) {
                addInstance(IcpType.INTERNAL_COUPLING, reference, "Internal coupling: $qualifiedName")
            } else if (!isJdkType(reference)) {
                addInstance(IcpType.EXTERNAL_COUPLING, reference, "External coupling: $qualifiedName")
            }
        }
        super.visitCtTypeReference(reference)
    }

    private fun isCouplingType(reference: CtTypeReference<*>): Boolean {
        if (reference.isPrimitive) return false
        if (reference.simpleName == "void") return false
        // Avoid duplicate counts in some cases like package references
        val parent = reference.parent
        return parent !is CtPackageReference
    }

    private fun isJdkType(reference: CtTypeReference<*>): Boolean {
        val qn = reference.qualifiedName
        if (qn.startsWith("java.") || qn.startsWith("javax.") || qn.startsWith("sun.")) return true

        // In no-classpath mode, if package is not resolved, check simple names for standard types
        val standardNames = setOf(
            "String",
            "List",
            "ArrayList",
            "Map",
            "HashMap",
            "Set",
            "HashSet",
            "Collection",
            "Integer",
            "Long",
            "Double",
            "Boolean",
            "Object"
        )
        return standardNames.contains(reference.simpleName) && (reference.`package` == null || reference.`package`.isImplicit)
    }

    private fun analyzeCondition(condition: CtExpression<Boolean>?) {
        if (condition == null) return

        // Initial condition
        addInstance(IcpType.CONDITION, condition, "condition expression")

        // Count logical operators (&&, ||)
        condition.accept(object : CtScanner() {
            override fun <T : Any?> visitCtBinaryOperator(operator: CtBinaryOperator<T>) {
                if (operator.kind == BinaryOperatorKind.AND || operator.kind == BinaryOperatorKind.OR) {
                    addInstance(IcpType.CONDITION, operator, "logical operator ${operator.kind}")
                }
                super.visitCtBinaryOperator(operator)
            }
        })
    }
}