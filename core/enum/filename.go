package coreenum

type CTXEnumExtension string

const (
	CTXEnumExtensionCSV  CTXEnumExtension = "csv"
	CTXEnumExtensionXLSX CTXEnumExtension = "xlsx"
	CTXEnumExtensionXLS  CTXEnumExtension = "xls"
)

var CTXEnumExtensionValues = []CTXEnumExtension{
	CTXEnumExtensionCSV,
	CTXEnumExtensionXLSX,
	CTXEnumExtensionXLS,
}

func (s CTXEnumExtension) IsValid() bool {
	switch s {
	case CTXEnumExtensionCSV, CTXEnumExtensionXLSX, CTXEnumExtensionXLS:
		return true
	}
	return false
}

func (s CTXEnumExtension) String() string {
	return string(s)
}
