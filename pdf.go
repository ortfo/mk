package ortfomk

import (
	"bytes"
	"fmt"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
)

func WritePDF(html string, to string) error {
	Status(StepGeneratePDF, ProgressDetails{
		File:     g.Progress.File,
		Language: g.CurrentLanguage,
	})
	generator, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return fmt.Errorf("could not initialize pdf generator: %w", err)
	}

	generator.AddPage(wkhtmltopdf.NewPageReader(bytes.NewBufferString(html)))
	generator.PageSize.Set(wkhtmltopdf.PageSizeA4)
	generator.Dpi.Set(300)

	err = generator.Create()
	if err != nil {
		return fmt.Errorf("could not create the pdf file: %w", err)
	}

	LogDebug("writing pdf to %s", to)
	err = generator.WriteFile(to)
	if err != nil {
		return fmt.Errorf("could not write the created pdf file to %q: %w", to, err)
	}

	return nil
}
