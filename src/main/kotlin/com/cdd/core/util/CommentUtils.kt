package com.cdd.core.util

object CommentUtils {
    /**
     * Strips a single-line comment (e.g., // ...) from a line of code.
     */
    fun stripLineComment(line: String, delimiter: String = "//"): String {
        val index = line.indexOf(delimiter)
        return if (index != -1) line.substring(0, index) else line
    }

    /**
     * Strips block comments (e.g., /* ... */) from a line of code.
     */
    fun stripBlockComments(line: String, startDelimiter: String = "/*", endDelimiter: String = "*/"): String {
        var result = line
        while (result.contains(startDelimiter) && result.contains(endDelimiter)) {
            val start = result.indexOf(startDelimiter)
            val end = result.indexOf(endDelimiter) + endDelimiter.length
            if (start < end) {
                result = result.removeRange(start, end)
            } else {
                break
            }
        }
        
        // Handle start without end on the same line
        if (result.contains(startDelimiter)) {
            val start = result.indexOf(startDelimiter)
            result = result.substring(0, start)
        }
        
        return result
    }

    /**
     * Checks if the line, after potentially stripping comments, contains any functional code.
     */
    fun hasCode(line: String, stripFn: (String) -> String): Boolean {
        if (line.trim().isEmpty()) return false
        val stripped = stripFn(line)
        return stripped.trim().isNotEmpty()
    }
}
