package billing

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/RamunnoAJ/aesovoy-server/internal/store"
	"github.com/xuri/excelize/v2"
	_ "golang.org/x/image/webp"
)

var (
	_, b, _, _   = runtime.Caller(0)
	projectRoot  = filepath.Join(filepath.Dir(b), "..", "..")
	templatePath = filepath.Join(projectRoot, "docs", "Plantilla.xlsx")
	invoiceDir   = filepath.Join(projectRoot, "facturas")
)

const (
	templateSheet = "Hoja1"
	itemsStartRow = 12
)

// GenerateInvoice creates or updates an Excel invoice file based on an order.
func GenerateInvoice(order *store.Order, client *store.Client, products map[int64]*store.Product) error {
	// 1. Determine filename
	dateStr := order.Date.Format("2006-01-02")
	fileName := fmt.Sprintf("remito_produccion-%s.xlsx", dateStr)
	filePath := filepath.Join(invoiceDir, fileName)

	var f *excelize.File
	var err error

	// 2. Check if file exists and open/create it
	if _, err = os.Stat(filePath); os.IsNotExist(err) {
		// File doesn't exist, create from template
		f, err = createInvoiceFileFromTemplate(filePath)
		if err != nil {
			return fmt.Errorf("could not create invoice file: %w", err)
		}
	} else {
		// File exists, open it
		f, err = excelize.OpenFile(filePath)
		if err != nil {
			return fmt.Errorf("could not open existing invoice file: %w", err)
		}
	}
	defer f.Close()

	// 3. Handle sheet
	sheetName := client.Name
	sheetIndex, err := f.GetSheetIndex(sheetName)
	if err != nil || sheetIndex == -1 {
		// Sheet doesn't exist, create it by cloning the template sheet
		templateIndex, err := f.GetSheetIndex(templateSheet)
		if err != nil || templateIndex == -1 {
			return fmt.Errorf("template sheet '%s' not found in template file", templateSheet)
		}

		newSheetIndex, err := f.NewSheet(sheetName)
		if err != nil {
			return fmt.Errorf("failed to create new sheet: %w", err)
		}

		err = f.CopySheet(templateIndex, newSheetIndex)
		if err != nil {
			// If copy fails, delete the newly created sheet before returning the error
			_ = f.DeleteSheet(sheetName)
			return fmt.Errorf("failed to copy sheet content: %w", err)
		}

		// Add logo to the new sheet from file bytes
		logoPath := filepath.Join(projectRoot, "docs", "logo.jpg")
		opts := &excelize.GraphicOptions{
			AutoFit: true,
		}
		for _, cell := range []string{"D1", "I1"} {
			err := f.AddPicture(sheetName, cell, logoPath, opts)
			if err != nil {
				log.Printf("could not add logo to cell %s: %v", cell, err)
			}
		}

		// Set header data on the new sheet
		setInvoiceHeaders(f, sheetName, order, client)
	}

	row := itemsStartRow
	var total float64 = 0

	for _, item := range order.Items {
		product, ok := products[item.ProductID]
		if !ok {
			continue
		}
		price, _ := strconv.ParseFloat(item.Price, 64)

		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), item.Quantity)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), product.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), price)

		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), item.Quantity)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), product.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), price)

		subtotal := float64(item.Quantity) * price
		total = total + subtotal
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), subtotal)
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), subtotal)

		row++
	}

	f.SetCellValue(sheetName, "D59", total)
	f.SetCellValue(sheetName, "I59", total)

	return f.Save()
}

func createInvoiceFileFromTemplate(filePath string) (*excelize.File, error) {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, err
	}

	// Open template
	templateFile, err := excelize.OpenFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("could not open template file: %w", err)
	}
	defer templateFile.Close()

	// Save as new file
	if err := templateFile.SaveAs(filePath); err != nil {
		return nil, err
	}

	// Re-open the new file to work with it
	return excelize.OpenFile(filePath)
}

func setInvoiceHeaders(f *excelize.File, sheetName string, order *store.Order, client *store.Client) {
	f.SetCellValue(sheetName, "B6", order.Date.Format("02/01/2006"))
	f.SetCellValue(sheetName, "B7", order.ID)
	f.SetCellValue(sheetName, "B8", client.Name)
	f.SetCellValue(sheetName, "B9", client.Address)

	f.SetCellValue(sheetName, "G6", order.Date.Format("02/01/2006"))
	f.SetCellValue(sheetName, "G7", order.ID)
	f.SetCellValue(sheetName, "G8", client.Name)
	f.SetCellValue(sheetName, "G9", client.Address)
}
