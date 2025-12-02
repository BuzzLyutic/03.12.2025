package pdf

import (
	"bytes"
	"fmt"

	"github.com/jung-kurt/gofpdf"
	"github.com/BuzzLyutic/03.12.2025/internal/model"
)

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) GenerateReport(linkSets []*model.LinkSet) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()

	// Заголовок
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(0, 12, "Links Status Report", "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Содержимое
	for _, set := range linkSets {
		pdf.SetFont("Arial", "B", 14)
		pdf.SetFillColor(230, 230, 230)
		pdf.CellFormat(0, 10, fmt.Sprintf("Set #%d", set.ID), "1", 1, "L", true, 0, "")

		pdf.SetFont("Arial", "", 11)
		for url, status := range set.Links {
			// Цвет статуса
			if status == "available" {
				pdf.SetTextColor(0, 128, 0) // Зелёный
			} else {
				pdf.SetTextColor(255, 0, 0) // Красный
			}

			pdf.CellFormat(130, 8, url, "LB", 0, "L", false, 0, "")
			pdf.CellFormat(0, 8, status, "RB", 1, "R", false, 0, "")
		}

		pdf.SetTextColor(0, 0, 0)
		pdf.Ln(5)
	}

	// Генерация PDF в буфер
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return buf.Bytes(), nil
}
