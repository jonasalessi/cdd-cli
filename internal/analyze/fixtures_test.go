package analyze

import "github.com/jonasalessi/cdd-cli/internal/config"

// The pipeline is language-agnostic, so its tests run against synthetic
// languages: a real id here would tie the package to the registry.
const langAlpha config.Language = "alpha"

// kindClass is the unit kind the fixtures report.
const kindClass = "class"
