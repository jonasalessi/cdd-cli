package com.cdd.core.config

import io.kotest.core.spec.style.DescribeSpec
import io.kotest.matchers.maps.shouldNotBeEmpty
import io.kotest.matchers.shouldBe
import java.io.File
import java.nio.file.Files

class ConfigurationManagerTest : DescribeSpec({
    val tempDir = Files.createTempDirectory("cdd-test").toFile()

    afterSpec {
        tempDir.deleteRecursively()
    }

    describe("ConfigurationManager") {
        it("should return default config when no file exists") {
            val config = ConfigurationManager.loadConfig(tempDir)
            config.metrics.shouldNotBeEmpty()
            config.sloc.methodLimit shouldBe 24
        }

        it("should load valid YAML configuration") {
            val yamlContent = """
                metrics:
                  java:
                    ".*":
                       code_branch: 15.0
                icp-limits:
                  java:
                    ".*": 12.0
                sloc:
                  methodLimit: 30
            """.trimIndent()
            val yamlFile = File(tempDir, ".cdd.yml")
            yamlFile.writeText(yamlContent)

            val config = ConfigurationManager.loadConfig(tempDir)
            val javaMetrics = config.metrics["java"]?.get(".*")
            javaMetrics?.get("code_branch") shouldBe 15.0
            config.sloc.methodLimit shouldBe 30
            config.icpLimits.shouldNotBeEmpty()
            config.icpLimits["java"]!![".*"] shouldBe 12.0

            yamlFile.delete()
        }

        it("should accept 0 as a valid limit") {
            val yamlContent = """
                metrics:
                  java:
                    ".*":
                       code_branch: 0.0
            """.trimIndent()
            val yamlFile = File(tempDir, ".cdd.yml")
            yamlFile.writeText(yamlContent)

            val config = ConfigurationManager.loadConfig(tempDir)
            val javaMetrics = config.metrics["java"]?.get(".*")
            javaMetrics?.get("code_branch") shouldBe 0.0

            yamlFile.delete()
        }

        it("should allow 0 as a valid metric weight") {
            val yamlContent = """
                metrics:
                  java:
                    ".*":
                      code_branch: 0
            """.trimIndent()

            val yamlFile = File(tempDir, ".cdd.yml")
            yamlFile.writeText(yamlContent)
            val config = ConfigurationManager.loadConfig(tempDir)
            config.metrics["java"]!![".*"]!!["code_branch"] shouldBe 0.0
            yamlFile.delete()
        }

        it("should allow 0 as a valid ICP limit") {
            val yamlContent = """
                icp-limits:
                  java:
                    ".*": 0
            """.trimIndent()

            val yamlFile = File(tempDir, ".cdd.yml")
            yamlFile.writeText(yamlContent)
            val config = ConfigurationManager.loadConfig(tempDir)
            config.icpLimits["java"]!![".*"] shouldBe 0.0
            yamlFile.delete()
        }

        it("should fallback to defaults if parsing fails") {
            val invalidYaml = "invalid: : yaml"
            val yamlFile = File(tempDir, ".cdd.yml")
            yamlFile.writeText(invalidYaml)

            val config = ConfigurationManager.loadConfig(tempDir)
            config.metrics.shouldNotBeEmpty() // defaults

            yamlFile.delete()
        }

        it("should fallback to defaults if validation fails") {
            val invalidLimit = """
                metrics:
                  java:
                    ".*":
                       code_branch: -1.0
            """.trimIndent()
            val yamlFile = File(tempDir, ".cdd.yml")
            yamlFile.writeText(invalidLimit)

            val config = ConfigurationManager.loadConfig(tempDir)
            // Should be defaults because validation failed
            config.metrics["java"]?.get(".*")?.get("code_branch") shouldBe 1.0

            yamlFile.delete()
        }
        it("should merge partial YAML with default values") {
            val yamlContent = """
                internal_coupling:
                  auto_detect: false
                  packages: [ "com.challenge" ]
            """.trimIndent()
            val yamlFile = File(tempDir, ".cdd.yml")
            yamlFile.writeText(yamlContent)

            val config = ConfigurationManager.loadConfig(tempDir)

            config.internalCoupling.autoDetect shouldBe false
            config.internalCoupling.packages shouldBe listOf("com.challenge")

            config.metrics.shouldNotBeEmpty()
            config.metrics["java"]!![".*"]!!["code_branch"] shouldBe 1.0
            config.metrics["kotlin"]!![".*"]!!["condition"] shouldBe 1.0

            config.icpLimits.shouldNotBeEmpty()
            config.icpLimits["java"]!![".*"] shouldBe 12.0

            yamlFile.delete()
        }
    }
})
