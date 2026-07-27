package coreenum

type CTXEnumRole string

const (
	CTXEnumRoleAdmin CTXEnumRole = "ADMIN"
	CTXEnumRoleUser  CTXEnumRole = "USER"
)

var CTXEnumRoleValues = []CTXEnumRole{
	CTXEnumRoleAdmin,
	CTXEnumRoleUser,
}

func (s *CTXEnumRole) IsValid() bool {
	switch *s {
	case CTXEnumRoleAdmin, CTXEnumRoleUser:
		return true
	}
	return false
}

func (s *CTXEnumRole) String() string {
	return string(*s)
}
