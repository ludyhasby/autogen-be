package coreenum

type CTXEnumLocationType string

const (
	CTXEnumLocationTypeCustomer    CTXEnumLocationType = "CUSTOMER"
	CTXEnumLocationTypeMeterPoint  CTXEnumLocationType = "METERPOINT"
	CTXEnumLocationTypePreCustomer CTXEnumLocationType = "PRE CUSTOMER"
)

var CTXEnumLocationTypeValues = []CTXEnumLocationType{
	CTXEnumLocationTypeCustomer,
	CTXEnumLocationTypeMeterPoint,
	CTXEnumLocationTypePreCustomer,
}

func (s *CTXEnumLocationType) IsValid() bool {
	switch *s {
	case CTXEnumLocationTypeCustomer, CTXEnumLocationTypeMeterPoint, CTXEnumLocationTypePreCustomer:
		return true
	}
	return false
}

func (s *CTXEnumLocationType) String() string {
	return string(*s)
}
