package coreenum

type CTXEnumVoltageType string

const (
	CTXEnumVoltageTypeTM CTXEnumVoltageType = "TM"
	CTXEnumVoltageTypeTR CTXEnumVoltageType = "TR"
)

var CTXEnumVoltageTypeValues = []CTXEnumVoltageType{
	CTXEnumVoltageTypeTM,
	CTXEnumVoltageTypeTR,
}

func (s *CTXEnumVoltageType) IsValid() bool {
	switch *s {
	case CTXEnumVoltageTypeTM, CTXEnumVoltageTypeTR:
		return true
	}
	return false
}

func (s *CTXEnumVoltageType) String() string {
	return string(*s)
}
