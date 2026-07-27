package coreenum

type CTXEnumStageProcess string

const (
	CTXEnumStageProcessConfigSetting CTXEnumStageProcess = "CONFIG_SETTING"
	CTXEnumStageProcessWeightSetting CTXEnumStageProcess = "CONFIG_WEIGHT"
	CTXEnumStageProcessReady         CTXEnumStageProcess = "READY"
	CTXEnumStageProcessDone          CTXEnumStageProcess = "DONE"
)

var CTXEnumStageProcessValues = []CTXEnumStageProcess{
	CTXEnumStageProcessConfigSetting,
	CTXEnumStageProcessWeightSetting,
	CTXEnumStageProcessReady,
	CTXEnumStageProcessDone,
}

func (s *CTXEnumStageProcess) IsValid() bool {
	switch *s {
	case CTXEnumStageProcessConfigSetting, CTXEnumStageProcessWeightSetting, CTXEnumStageProcessReady, CTXEnumStageProcessDone:
		return true
	}
	return false
}

func (s *CTXEnumStageProcess) ConvertToStr() string {
	switch *s {
	case CTXEnumStageProcessConfigSetting:
		return "konfigurasi nilai parameter"
	case CTXEnumStageProcessWeightSetting:
		return "konfigurasi bobot parameter"
	case CTXEnumStageProcessReady:
		return "siap diproses"
	case CTXEnumStageProcessDone:
		return "sudah selesai diproses"
	}
	return "tidak diketahui"
}

func (s *CTXEnumStageProcess) String() string {
	return string(*s)
}
