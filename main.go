package main

import (
	"github.com/poblish/boulevard/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(analysis.DashboardGenerator)
}
