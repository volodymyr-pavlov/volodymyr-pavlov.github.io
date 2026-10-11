package gen

import (
	"fmt"

	"github.com/evanw/esbuild/pkg/api"
)

// bundleScript bundles the script at entry with every module it imports into one minified script, which drops the
// whitespace and comments and shortens the local names. Everything a page can reach, such as element ids and data
// attributes, is left as written.
func bundleScript(entry string) ([]byte, error) {
	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{entry},
		Bundle:            true,
		Write:             false,
		Format:            api.FormatIIFE,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		LegalComments:     api.LegalCommentsNone,
		Target:            api.ES2017,
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
