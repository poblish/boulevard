package generation

import (
	"fmt"
	"log"
	"os"
	"text/template"
)

const AlertMetricsTemplate = `boulevard-alert-rules: "{{ .AlertsCount }}"
boulevard-unique-metrics: "{{ .UniqueMetricsCount }}"`

type AlertMetrics struct {
	Count int
}

type AlertMetricsOutput struct {
	AlertsCount        int
	UniqueMetricsCount int
}

func (a AlertMetricsOutput) WriteToFile(outputpath string) {
	fmt.Println("Writing metric labels to", outputpath)

	tmpl, _ := template.New("default").Parse(AlertMetricsTemplate)

	outputFile, err := os.Create(outputpath)
	if err != nil {
		log.Fatalf("Metrics file creation failed: %s", err)
	}

	defer outputFile.Close()

	tErr := tmpl.Execute(outputFile, &a)
	if tErr != nil {
		log.Fatalf("Metrics file template execution: %s", tErr)
	}
}
