package coreenum

type CTXEnumUseCase string

const (
	CTXEnumUseCaseAMR        CTXEnumUseCase = "AMR"
	CTXEnumUseCasePascaBayar CTXEnumUseCase = "PASCABAYAR"
	CTXEnumUseCasePrabayar   CTXEnumUseCase = "PRABAYAR"
)

func (s *CTXEnumUseCase) IsValid() bool {
	switch *s {
	case CTXEnumUseCaseAMR, CTXEnumUseCasePascaBayar, CTXEnumUseCasePrabayar:
		return true
	}
	return false
}

func (s *CTXEnumUseCase) String() string {
	return string(*s)
}
