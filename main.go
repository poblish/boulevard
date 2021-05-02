package main

import (
	"github.com/poblish/boulevard/generation"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(generation.DashboardGenerator)
}
