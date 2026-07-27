package coreenum

type CTXEnumMeasurementType string

const (
	CTXEnumMeasurementTypeLangsung    CTXEnumMeasurementType = "LANGSUNG"
	CTXEnumMeasurementTypeTakLangsung CTXEnumMeasurementType = "TAK_LANGSUNG"
)

var CTXEnumMeasurementTypeValues = []CTXEnumMeasurementType{
	CTXEnumMeasurementTypeLangsung,
	CTXEnumMeasurementTypeTakLangsung,
}

func (s *CTXEnumMeasurementType) IsValid() bool {
	switch *s {
	case CTXEnumMeasurementTypeLangsung, CTXEnumMeasurementTypeTakLangsung:
		return true
	}
	return false
}

func (s *CTXEnumMeasurementType) String() string {
	return string(*s)
}
