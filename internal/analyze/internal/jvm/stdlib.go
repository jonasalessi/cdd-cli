package jvm

import "strings"

// jdkPrefixes are the package prefixes the JDK ships, written with their
// trailing dot so that a prefix matches the packages below it and never a
// longer name beside it ("javax." must not match "javassist").
//
// The javax packages are listed one by one rather than taken as a whole
// prefix: the same prefix carries libraries that never shipped with the
// platform -- javax.servlet, javax.persistence, javax.inject -- and coupling
// to those is coupling to a third party.
var jdkPrefixes = []string{
	"java.",
	"jdk.",
	"javax.accessibility.",
	"javax.annotation.processing.",
	"javax.crypto.",
	"javax.imageio.",
	"javax.lang.model.",
	"javax.management.",
	"javax.naming.",
	"javax.net.",
	"javax.print.",
	"javax.rmi.ssl.",
	"javax.script.",
	"javax.security.auth.",
	"javax.security.cert.",
	"javax.security.sasl.",
	"javax.smartcardio.",
	"javax.sound.",
	"javax.sql.",
	"javax.swing.",
	"javax.tools.",
	"javax.transaction.xa.",
	"javax.xml.",
}

// IsJDK reports whether a qualified path names a package the JDK ships, which
// is what the JVM languages call their standard library. A path that equals a
// prefix counts, as it does in IsInternal.
func IsJDK(path string) bool {
	for _, prefix := range jdkPrefixes {
		if strings.HasPrefix(path, prefix) || path+"." == prefix {
			return true
		}
	}
	return false
}
