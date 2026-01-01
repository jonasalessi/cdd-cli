package com.cdd.core.config

import com.charleskorn.kaml.Yaml
import org.slf4j.LoggerFactory
import java.io.File

/**
 * Manages loading and validation of CDD configuration files.
 */
object ConfigurationManager {
    private val logger = LoggerFactory.getLogger(ConfigurationManager::class.java)

    private val yaml = Yaml.default

    fun loadConfig(workingDir: File): CddConfig {
        val ymlFile = File(workingDir, ".cdd.yml")
        val yamlFile = File(workingDir, ".cdd.yaml")
        if (ymlFile.exists()) {
            return tryLoad(ymlFile) { yaml.decodeFromString(CddConfig.serializer(), it) }
        }
        if (yamlFile.exists()) {
            return tryLoad(yamlFile) { yaml.decodeFromString(CddConfig.serializer(), it) }
        }
        logger.info("No configuration file found in ${workingDir.absolutePath}. Using defaults.")
        return createDefaultConfig()
    }

    private fun createDefaultConfig(): CddConfig {
        val defaultMetrics = mapOf(
            "code_branch" to 1.0,
            "condition" to 1.0,
            "internal_coupling" to 1.0,
            "exception_handling" to 1.0
        )
        val defaultLimits = mapOf(".*" to 12.0)
        return CddConfig(
            metrics = mapOf(
                "java" to mapOf(".*" to defaultMetrics),
                "kotlin" to mapOf(".*" to defaultMetrics)
            ),
            icpLimits = mapOf(
                "java" to defaultLimits,
                "kotlin" to defaultLimits
            )
        )
    }

    /**
     * Loads configuration from a specific file.
     * Returns default configuration if the file does not exist or if parsing fails.
     */
    fun loadConfigFile(file: File): CddConfig {
        if (!file.exists()) {
            logger.error("Configuration file not found: ${file.absolutePath}. Using defaults.")
            return createDefaultConfig()
        }

        return when (file.extension.lowercase()) {
            "yml", "yaml" -> tryLoad(file) { yaml.decodeFromString(CddConfig.serializer(), it) }
            else -> {
                logger.error("Unsupported configuration file format: ${file.extension}. Expected .yml or .yaml. Using defaults.")
                createDefaultConfig()
            }
        }
    }

    private fun tryLoad(file: File, decoder: (String) -> CddConfig): CddConfig {
        return try {
            val content = file.readText()
            val config = decoder(content)
            validate(config)
            config
        } catch (e: Exception) {
            logger.error("Failed to load configuration from ${file.name}: ${e.message}. Using defaults.")
            createDefaultConfig()
        }
    }

    private fun validate(config: CddConfig) {
        config.metrics.forEach { (lang, patterns) ->
            patterns.forEach { (pattern, metrics) ->
                 metrics.forEach { (metric, value) ->
                     require(value >= 0) { "Weight for $metric in $lang:$pattern must be greater than or equal to 0" }
                 }
            }
        }
        config.icpLimits.forEach { (lang, patterns) ->
            patterns.forEach { (pattern, value) ->
                require(value >= 0) { "Limit for $lang:$pattern must be greater than or equal to 0" }
            }
        }
    }
}
