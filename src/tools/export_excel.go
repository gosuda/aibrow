package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/xuri/excelize/v2"
	"github.com/yjlee/aibrow/src/store"
)

const exportExcelName = "export_excel"

// ExportExcelInput is the expected JSON input for export_excel.
type ExportExcelInput struct {
	ImageID string `json:"image_id"`
}

// ExportExcelResult is returned by export_excel.
type ExportExcelResult struct {
	ImageID string `json:"image_id"`
	Path    string `json:"path"`
}

// RegisterExportExcel registers the export_excel tool.
func RegisterExportExcel(r *Registry, store *store.Store, outputDir string) {
	r.Register(Tool{
		Name:        exportExcelName,
		Description: "Exports the image analysis and adjustment parameters to an Excel file.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"image_id": {"type": "string"}
			},
			"required": ["image_id"]
		}`),
		Execute: func(ctx context.Context, input json.RawMessage) (interface{}, error) {
			var in ExportExcelInput
			if err := json.Unmarshal(input, &in); err != nil {
				return nil, fmt.Errorf("export_excel: parse input: %w", err)
			}
			record, ok := store.Get(in.ImageID)
			if !ok {
				return nil, fmt.Errorf("export_excel: image %s not found", in.ImageID)
			}
			output := filepath.Join(outputDir, in.ImageID+"_preset.xlsx")
			if err := WritePresetExcel(output, record); err != nil {
				return nil, fmt.Errorf("export_excel: %w", err)
			}
			return ExportExcelResult{ImageID: in.ImageID, Path: output}, nil
		},
	})
}

// WritePresetExcel writes an Excel preset summary for the given record.
func WritePresetExcel(path string, record store.ImageRecord) error {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := "Preset"
	f.SetSheetName("Sheet1", sheet)

	_ = f.SetCellValue(sheet, "A1", "Field")
	_ = f.SetCellValue(sheet, "B1", "Value")

	rows := [][]interface{}{
		{"Image ID", record.ID},
		{"Original Path", record.OriginalPath},
		{"Processed Path", record.ProcessedPath},
		{"Width", record.Analysis.Width},
		{"Height", record.Analysis.Height},
		{"L Mean", record.Analysis.LMean},
		{"L Std", record.Analysis.LStd},
		{"C Mean", record.Analysis.CMean},
		{"C Std", record.Analysis.CStd},
		{"H Mean", record.Analysis.HMean},
		{"Shadow Lift", record.Analysis.ShadowLift},
		{"Highlight Gain", record.Analysis.HighlightGain},
		{"Black Level", record.Analysis.BlackLevel},
		{"White Level", record.Analysis.WhiteLevel},
		{"Micro Contrast", record.Analysis.MicroContrast},
		{"Brightness", record.Params.Brightness},
		{"Contrast", record.Params.Contrast},
		{"Saturation", record.Params.Saturation},
		{"Gamma", record.Params.Gamma},
		{"Micro Contrast Param", record.Params.MicroContrast},
		{"Temperature", record.Params.Temperature},
	}

	for i, row := range rows {
		cellA, _ := excelize.CoordinatesToCellName(1, i+2)
		cellB, _ := excelize.CoordinatesToCellName(2, i+2)
		_ = f.SetCellValue(sheet, cellA, row[0])
		_ = f.SetCellValue(sheet, cellB, row[1])
	}

	return f.SaveAs(path)
}
