package pdf

import (
	"fmt"
	"paymentfc/models"
	"time"

	"github.com/jung-kurt/gofpdf"
)

func GenerateInvoicePdf(payment *models.Payment, outputPath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "arial")
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(40, 10, "GO-Commerce Invoice Details")

	pdf.Ln(20)
	pdf.SetFont("Arial", "B", 12)

	pdf.Cell(40, 10, fmt.Sprintf("Invoice Date: %s", time.Now().Format("2006-01-02")))
	pdf.Ln(20)
	pdf.Cell(40, 10, fmt.Sprintf("Payment ID: %d", payment.ID))
	pdf.Ln(20)
	pdf.Cell(40, 10, fmt.Sprintf("Order ID: %d", payment.OrderID))
	pdf.Ln(20)
	pdf.Cell(40, 10, fmt.Sprintf("User ID: %d", payment.UserID))
	pdf.Ln(20)
	pdf.Cell(40, 10, fmt.Sprintf("Amount: %f", payment.Amount))
	pdf.Ln(20)
	pdf.Cell(40, 10, fmt.Sprintf("Status: %s", payment.Status))
	pdf.Ln(20)
	pdf.Cell(40, 10, fmt.Sprintf("Create Time: %s", payment.CreateTime))
	pdf.Ln(20)
	pdf.Cell(40, 10, fmt.Sprintf("Expired Time: %s", payment.ExpiredTime))
	pdf.Ln(20)

	return pdf.OutputFileAndClose(outputPath)
}
