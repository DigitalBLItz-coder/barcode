package main

import (
	"fmt"
	"html/template"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"strconv"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/fogleman/gg"
)

const (
	baseDPI = 96
)

// BarcodeData stores properties of each barcode
type BarcodeData struct {
	Data         string
	Width        int
	Height       int
	PaddingColor string
	FontChoice   string
	TextColor    string
	TextSize     int
	Bold         bool
}

// Parse HEX color to color.RGBA
func parseHexColor(s string) (color.Color, error) {
	c, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return nil, err
	}
	return color.RGBA{uint8(c >> 16), uint8(c >> 8 & 0xFF), uint8(c & 0xFF), 0xFF}, nil
}

// Handle barcode generation
func generateBarcode(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	var barcodes []BarcodeData

	for i := 1; i <= 4; i++ {
		data := r.FormValue(fmt.Sprintf("data%d", i))
		if data == "" {
			continue
		}

		width, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("width%d", i)))
		height, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("height%d", i)))
		paddingColor := r.FormValue(fmt.Sprintf("padding_color%d", i))
		fontChoice := r.FormValue(fmt.Sprintf("font_choice%d", i))
		textColor := r.FormValue(fmt.Sprintf("text_color%d", i))
		textSize, _ := strconv.Atoi(r.FormValue(fmt.Sprintf("text_size%d", i)))
		bold := r.FormValue(fmt.Sprintf("bold%d", i)) == "on"

		barcodes = append(barcodes, BarcodeData{
			Data:         data,
			Width:        width,
			Height:       height,
			PaddingColor: paddingColor,
			FontChoice:   fontChoice,
			TextColor:    textColor,
			TextSize:     textSize,
			Bold:         bold,
		})
	}

	if len(barcodes) == 0 {
		http.Error(w, "No barcode data provided", http.StatusBadRequest)
		return
	}

	totalWidth := 0
	totalHeight := 0

	// Calculate total canvas size based on barcodes and padding
	for _, barcode := range barcodes {
		totalWidth += barcode.Width + (barcode.TextSize * 2) // Side padding based on text size
		if barcode.Height+(barcode.TextSize*3) > totalHeight {
			totalHeight = barcode.Height + (barcode.TextSize * 3) // Adaptive bottom padding based on text size
		}
	}

	dc := gg.NewContext(totalWidth, totalHeight)
	xOffset := 0

	for _, b := range barcodes {
		// Parse padding color
		paddingColor, err := parseHexColor(b.PaddingColor)
		if err != nil {
			http.Error(w, "Invalid padding color", http.StatusBadRequest)
			return
		}

		// Create the barcode
		bar, err := code128.Encode(b.Data)
		if err != nil {
			http.Error(w, "Failed to generate barcode", http.StatusInternalServerError)
			return
		}

		// Scale barcode to desired width and height (96 DPI)
		widthAtDPI := b.Width * baseDPI / 96
		heightAtDPI := b.Height * baseDPI / 96
		scaledBar, err := barcode.Scale(bar, widthAtDPI, heightAtDPI)
		if err != nil {
			http.Error(w, "Failed to scale barcode", http.StatusInternalServerError)
			return
		}

		// Draw background (padding color)
		dc.SetColor(paddingColor)
		dc.DrawRectangle(float64(xOffset), 0, float64(b.Width+(b.TextSize*2)), float64(totalHeight))
		dc.Fill()

		// Draw barcode image
		dc.DrawImage(scaledBar, xOffset+b.TextSize, b.TextSize)

		// Parse text color
		textColor, err := parseHexColor(b.TextColor)
		if err != nil {
			http.Error(w, "Invalid text color", http.StatusBadRequest)
			return
		}
		dc.SetColor(textColor)

		// Set font
		if b.Bold {
			dc.LoadFontFace("static/arial_black.ttf", float64(b.TextSize))
		} else {
			dc.LoadFontFace("static/arial.ttf", float64(b.TextSize))
		}

		// Draw the text centered under the barcode
		textX := float64(xOffset + b.TextSize + (b.Width / 2))
		textY := float64(b.TextSize + b.Height + b.TextSize)
		dc.DrawStringAnchored(b.Data, textX, textY, 0.5, 0.5)

		// Increment xOffset for the next barcode
		xOffset += b.Width + (b.TextSize * 2)
	}

	// Save the barcode image to a temporary path
	filePath := "static/generated_barcode.png"
	outFile, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "Failed to save barcode", http.StatusInternalServerError)
		return
	}
	png.Encode(outFile, dc.Image())
	outFile.Close()

	// Render the generated_barcode.html template
	tmpl, err := template.ParseFiles("templates/generated_barcode.html")
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	// Serve the page with the generated barcode path
	data := struct {
		BarcodePath string
	}{
		BarcodePath: filePath,
	}
	tmpl.Execute(w, data)
}

// Serve the form
func serveForm(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "index.html")
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/", serveForm)
	http.HandleFunc("/barcode", generateBarcode)

	fmt.Println("Barcode Generator started Navigate to  http://localhost:8080 to generate")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Failed to start server:", err)
	}
}
