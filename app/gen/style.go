package gen

import (
	"fmt"

	"github.com/evanw/esbuild/pkg/api"
)

// bundleStyle bundles the stylesheet at entry with every stylesheet it imports into one minified stylesheet, which
// drops the whitespace and comments and shortens the values. Selectors, custom property names and URLs are left as
// written.
func bundleStyle(entry string) ([]byte, error) {
	result := api.Build(api.BuildOptions{
		EntryPoints:      []string{entry},
		Bundle:           true,
		Write:            false,
		MinifyWhitespace: true,
		MinifySyntax:     true,
		LegalComments:    api.LegalCommentsNone,
	})
	if len(result.Errors) > 0 {
		e := result.Errors[0]
		if e.Location != nil {
			return nil, fmt.Errorf("%s:%d: %s", e.Location.File, e.Location.Line, e.Text)
		}
		return nil, fmt.Errorf("%s", e.Text)
	}
	if len(result.OutputFiles) != 1 {
		return nil, fmt.Errorf("%s: bundled into %d files, want 1", entry, len(result.OutputFiles))
	}
	return result.OutputFiles[0].Contents, nil
}
