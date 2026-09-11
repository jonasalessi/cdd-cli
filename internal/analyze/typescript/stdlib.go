package typescript

import "strings"

// nodeBuiltins are the modules the Node.js runtime ships, importable without
// the "node:" prefix. Deno and Bun standard libraries are out of scope: they
// are imported from URLs or from "bun:" specifiers, which stay external.
var nodeBuiltins = map[string]bool{
	"assert": true, "async_hooks": true, "buffer": true, "child_process": true,
	"cluster": true, "console": true, "constants": true, "crypto": true,
	"dgram": true, "diagnostics_channel": true, "dns": true, "domain": true,
	"events": true, "fs": true, "http": true, "http2": true, "https": true,
	"inspector": true, "module": true, "net": true, "os": true, "path": true,
	"perf_hooks": true, "process": true, "punycode": true, "querystring": true,
	"readline": true, "repl": true, "stream": true, "string_decoder": true,
	"sys": true, "timers": true, "tls": true, "trace_events": true,
	"tty": true, "url": true, "util": true, "v8": true, "vm": true,
	"wasi": true, "worker_threads": true, "zlib": true,
}

// isBuiltin reports whether spec names a Node.js built-in module. The
// "node:" prefix is authoritative, so "node:test" and "node:sqlite" are
// covered without listing them; a bare specifier is a built-in when its
// first slash-delimited segment is one, which admits the submodules
// "fs/promises", "path/posix", "stream/web" and "timers/promises" while
// leaving "node-fetch" and "@types/node" external.
func isBuiltin(spec string) bool {
	if strings.HasPrefix(spec, "node:") {
		return true
	}
	root, _, _ := strings.Cut(spec, "/")
	return nodeBuiltins[root]
}
