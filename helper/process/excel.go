package helperprocess

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

var (
	ErrUnsupportedTemplate error = fmt.Errorf("file tidak sesuai dengan template")
	ErrExceedMaxData       error = fmt.Errorf("jumlah baris melebihi batas maksimum")
)

func ExtractAndValidateHeader(xlsx *excelize.File, headerExpectation []string) (rows *excelize.Rows, mapIndex map[string]int, err error) {
	sheets := xlsx.GetSheetList()
	if len(sheets) == 0 {
		err = ErrUnsupportedTemplate
		return
	}

	rows, err = xlsx.Rows(sheets[0])
	if err != nil {
		return
	}
	rows.Next()
	headerRow, err := rows.Columns()
	if err != nil {
		return
	}

	mapIndex = make(map[string]int)
	for _, header := range headerExpectation {
		mapIndex[header] = -1
	}

	for i, col := range headerRow {
		col = strings.ToUpper(strings.TrimSpace(col))
		if _, exists := mapIndex[col]; exists {
			mapIndex[col] = i
		}
	}
	for _, v := range mapIndex {
		if v == -1 {
			err = ErrUnsupportedTemplate
			return
		}
	}
	return
}

func AddEmptyRowErr(i int64, fieldName string) error {
	return fmt.Errorf("baris tidak boleh kosong, field %s pada baris ke %d kosong", fieldName, i)
}

func AddUnsupportFieldErr(i int64, colName string, message string) error {
	return fmt.Errorf("tipe data %s pada baris %d tidak sesuai. %s", colName, i, message)
}

func ParseFloatField(val string, rowIdx int64, colName string) (float64, error) {
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, AddUnsupportFieldErr(rowIdx, colName, "Numerik/float")
	}
	return f, nil
}

func ParseIntField(val string, rowIdx int64, colName string) (int, error) {
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, AddUnsupportFieldErr(rowIdx, colName, "Integer")
	}
	return n, nil
}

func CellGuard(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}
