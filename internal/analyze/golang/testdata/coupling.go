// coupling.go — with InternalPrefixes = ["example.com/app"]

package billing

import (
	"fmt"
	"net/http"
	str "strings"

	_ "embed"

	money "example.com/app/money/v2"
	"example.com/app/shared"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

type Invoice struct {
	ID     uuid.UUID    // local_variable 1, external_coupling 1 (uuid)
	Amount money.Amount // local_variable 1, internal_coupling 1 (money)
	Client *http.Client // local_variable 1, stdlib_coupling 1 (net/http)
}

func (i Invoice) Render(raw []byte) string {
	var cfg struct { // local_variable 1 (cfg)
		Name string // local_variable 1 — a field of an anonymous struct
	}
	_ = yaml.Unmarshal(raw, &cfg) // external_coupling 1 (yaml)
	shared.Format(cfg.Name)       // internal_coupling 1 (shared)
	return fmt.Sprint(i.ID, str.ToUpper(cfg.Name))
}

type Note struct{}

func (Note) Text(fmt interface{ Sprint() string }) string {
	return fmt.Sprint() // stdlib_coupling 0 — the parameter shadows the package
}

type Plain struct{}

var _ = errgroup.Group{}

// unit Invoice: internal_coupling 2 (money, shared), external_coupling 2
//   (uuid, yaml), stdlib_coupling 4 (fmt, net/http, strings, embed),
//   local_variable 5
// unit Note: 0 / 0 / 1 — only the blank `embed` import, which charges every
//   unit; `fmt` here is a parameter, so its Ident resolves and is not the
//   package
// unit Plain: 0 / 0 / 1 — same blank import, nothing else
// `golang.org/x/sync/errgroup` is external and used only by a top-level var,
//   so it is charged to no unit at all.
// Every coupling occurrence sits on its ImportSpec line, above unit Invoice.
