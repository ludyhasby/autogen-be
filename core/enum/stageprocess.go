package coreenum

type CTXEnumStageProcess string

const (
	CTXEnumStageProcessQueue         CTXEnumStageProcess = "QUEUE"
	CTXEnumStageProcessConfigSetting CTXEnumStageProcess = "CONFIG_SETTING"
	CTXEnumStageProcessWeightSetting CTXEnumStageProcess = "CONFIG_WEIGHT"
	CTXEnumStageProcessReady         CTXEnumStageProcess = "READY"
	CTXEnumStageProcessDone          CTXEnumStageProcess = "DONE"
	CTXEnumStageProcessProcessing    CTXEnumStageProcess = "PROCESSING"
	CTXEnumStageProcessFailed        CTXEnumStageProcess = "FAILED"
)

var CTXEnumStageProcessValues = []CTXEnumStageProcess{
	CTXEnumStageProcessQueue,
	CTXEnumStageProcessConfigSetting,
	CTXEnumStageProcessWeightSetting,
	CTXEnumStageProcessReady,
	CTXEnumStageProcessDone,
	CTXEnumStageProcessProcessing,
	CTXEnumStageProcessFailed,
}

func (s *CTXEnumStageProcess) IsValid() bool {
	switch *s {
	case CTXEnumStageProcessQueue, CTXEnumStageProcessConfigSetting, CTXEnumStageProcessWeightSetting, CTXEnumStageProcessReady, CTXEnumStageProcessDone, CTXEnumStageProcessProcessing, CTXEnumStageProcessFailed:
		return true
	}
	return false
}

func (s *CTXEnumStageProcess) ConvertToStr() string {
	switch *s {
	case CTXEnumStageProcessQueue:
		return "dalam antrian"
	case CTXEnumStageProcessConfigSetting:
		return "konfigurasi nilai parameter"
	case CTXEnumStageProcessWeightSetting:
		return "konfigurasi bobot parameter"
	case CTXEnumStageProcessReady:
		return "siap diproses"
	case CTXEnumStageProcessDone:
		return "sudah selesai diproses"
	case CTXEnumStageProcessProcessing:
		return "sedang dalam proses"
	case CTXEnumStageProcessFailed:
		return "gagal di proses"
	}
	return "tidak diketahui"
}

func (s *CTXEnumStageProcess) String() string {
	return string(*s)
}
