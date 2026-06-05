package main

import (
	"fmt"
	"net/http"
	"os"

	"goblog/internal/generator"
)

const defaultPort = "8080"
const outputDir = "public"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "build":
		build()

	case "serve":
		port := defaultPort
		if len(os.Args) >= 3 {
			port = os.Args[2]
		}
		// Build first so the server always has up-to-date content.
		fmt.Println("Building site...")
		build()
		serve(port)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func serve(port string) {
	fmt.Printf("Serving on http://localhost:%s  (Ctrl+C to stop)\n", port)
	fs := http.FileServer(http.Dir(outputDir))
	if err := http.ListenAndServe(":"+port, fs); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func build() {
	if err := generator.Build(".", outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Build error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Site built successfully → public/")
}

func printUsage() {
	fmt.Println("goblog - Static Site Generator")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run main.go <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  build           Convert posts/ to static HTML in public/")
	fmt.Println("  serve [port]    Build then serve public/ over HTTP (default port: 8080)")
}
