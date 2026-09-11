package jvm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestIsJDK (TC-J1, TC-J2, TC-J3) is the platform table: the JDK trees and
// the javax packages it ships are in, the libraries sharing the javax prefix
// are out, and a bare or lookalike package matches nothing.
func TestIsJDK(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"java.util.List", true},
		{"java.time.Instant", true},
		{"jdk.incubator.vector.X", true},
		{"javax.crypto.Cipher", true},
		{"javax.xml.parsers.DocumentBuilder", true},
		{"javax.swing.JFrame", true},
		{"javax.annotation.processing.Processor", true},
		{"javax.crypto", true},
		{"javax.inject.Inject", false},
		{"javax.servlet.http.HttpServlet", false},
		{"javax.persistence.Entity", false},
		{"javax.annotation.Nullable", false},
		{"javafx.scene.Node", false},
		{"javassist.ClassPool", false},
		{"org.springframework.stereotype.Service", false},
		{"kotlin.collections.List", false},
		{"javaxfoo.Bar", false},
		{"javax", false},
		{"", false},
	}
	for _, c := range cases {
		require.Equal(t, c.want, IsJDK(c.path), c.path)
	}
}
