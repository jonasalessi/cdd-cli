package com.cdd.core.config

import com.charleskorn.kaml.Yaml
import kotlinx.serialization.json.Json
import java.io.File
import org.slf4j.LoggerFactory

/**
 * Manages loading and validation of CDD configuration files.
 */
object ConfigurationManager {
    private val logger = LoggerFactory.getLogger(ConfigurationManager::class.java)
    
    // Explicitly configure YAML to be lenient
    private val yaml = Yaml.default
    
    // Explicitly configure JSON to be lenient
    private val json = Json {
        ignoreUnknownKeys = true
        isLenient = true
    }

    /**
     * Loads configuration from a directory, looking for .cdd.yml or .cdd.json.
     * Returns default configuration if no file is found or if parsing fails (with error logged).
     */
    fun loadConfig(workingDir: File): CddConfig {
        val ymlFile = File(workingDir, ".cdd.yml")
        val yamlFile = File(workingDir, ".cdd.yaml")
        if (ymlFile.exists()) {
            return tryLoad(ymlFile) { yaml.decodeFromString(CddConfig.serializer(), it) }
        } else if (yamlFile.exists()) {
            return tryLoad(yamlFile) { yaml.decodeFromString(CddConfig.serializer(), it) }
        }

        val jsonFile = File(workingDir, ".cdd.json")
        if (jsonFile.exists()) {
            return tryLoad(jsonFile) { json.decodeFromString(CddConfig.serializer(), it) }
        }

        logger.info("No configuration file found in ${workingDir.absolutePath}. Using defaults.")
        return CddConfig()
    }

    /**
     * Loads configuration from a specific file.
     * Returns default configuration if the file does not exist or if parsing fails.
     */
    fun loadConfigFile(file: File): CddConfig {
        if (!file.exists()) {
            logger.error("Configuration file not found: ${file.absolutePath}. Using defaults.")
            return CddConfig()
        }

        return when (file.extension.lowercase()) {
            "yml", "yaml" -> tryLoad(file) { yaml.decodeFromString(CddConfig.serializer(), it) }
            "json" -> tryLoad(file) { json.decodeFromString(CddConfig.serializer(), it) }
            else -> {
                logger.error("Unsupported configuration file format: ${file.extension}. Expected .yml or .json. Using defaults.")
                CddConfig()
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
            CddConfig()
        }
    }

    private fun validate(config: CddConfig) {
        require(config.limit > 0) { "ICP limit must be greater than 0" }
        require(config.sloc.methodLimit >= 0) { "SLOC method limit cannot be negative" }
        require(config.sloc.classLimit >= 0) { "SLOC class limit cannot be negative" }
        
        config.icpTypes.forEach { (type, weight) ->
            require(weight >= 0) { "Weight for $type cannot be negative" }
        }
    }
}
