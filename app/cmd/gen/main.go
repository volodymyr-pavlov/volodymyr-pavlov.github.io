// Command gen builds the blog with the gen package, and can serve the result for a preview.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/volodymyr-pavlov/volodymyr-pavlov.github.io/app/gen"
)

func main() {
	var opts gen.Options
	var port int
	flag.StringVar(&opts.Web, "web", "web", "directory of static pages and assets")
	flag.StringVar(&opts.Blog, "blog", "blog", "directory of article Markdown")
	flag.StringVar(&opts.Out, "out", ".dist", "output directory")
	flag.StringVar(&opts.Config, "config", "site.json", "site configuration")
	flag.IntVar(&port, "serve", 0, "after building, serve the output on this port")
	flag.Parse()

	report, err := gen.Run(opts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(report.Summary(opts.Out))
	if port != 0 {
		log.Fatal(gen.Serve(opts.Out, port))
	}
}
