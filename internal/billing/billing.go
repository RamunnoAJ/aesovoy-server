package billing

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
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
	sheetName := getSheetName(order.ID, client.Name)
	
	// If sheet exists, delete it to ensure clean state (e.g. if order items changed)
	if idx, err := f.GetSheetIndex(sheetName); err == nil && idx != -1 {
		f.DeleteSheet(sheetName)
	}

	// Create sheet from template
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
		_ = f.DeleteSheet(sheetName)
		return fmt.Errorf("failed to copy sheet content: %w", err)
	}

	// Add logo
	logoPath := filepath.Join(projectRoot, "docs", "logo.jpg")
	opts := &excelize.GraphicOptions{AutoFit: true}
	for _, cell := range []string{"D1", "I1"} {
		if err := f.AddPicture(sheetName, cell, logoPath, opts); err != nil {
			log.Printf("could not add logo to cell %s: %v", cell, err)
		}
	}

	// Set headers
	setInvoiceHeaders(f, sheetName, order, client)

	row := itemsStartRow
	var total float64 = 0

	log.Printf("Generating invoice for Order #%d. Items count: %d. Products map size: %d", order.ID, len(order.Items), len(products))

	for _, item := range order.Items {
		product, ok := products[item.ProductID]
		if !ok {
			log.Printf("Product ID %d not found in products map for Order Item", item.ProductID)
			continue
		}
		price, _ := strconv.ParseFloat(item.Price, 64)

		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), item.Quantity); err != nil {
			log.Printf("Error setting cell A%d: %v", row, err)
		}
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

func getSheetName(orderID int64, clientName string) string {
	// Format: "ID-ClientName"
	// Max length 31.
	// ID can be e.g. 5 digits. "12345-" takes 6 chars.
	// We should safe truncate.
	name := fmt.Sprintf("%d-%s", orderID, clientName)
	if len(name) > 31 {
		return name[:31]
	}
	return name
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
