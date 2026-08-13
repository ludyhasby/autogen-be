package helperprocess

import "github.com/xuri/excelize/v2"

func Style(fileExcel *excelize.File) (titleStyle int, alignCenter int) {
	titleStyle, _ = fileExcel.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size: 15,
			Bold: true,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#87CEEB"},
			Pattern: 1,
		},
	})
	alignCenter, _ = fileExcel.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	return
}
