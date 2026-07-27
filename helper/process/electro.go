package helperprocess

import (
	coreenum "logisfy/core/enum"
	"math"
)

const eps = 0.001

func CalculatePowerFactor(activePower, apparentPower, currentAngle, voltageAngle float64) float64 {
	if apparentPower != 0 {
		return activePower / apparentPower
	}

	angle := currentAngle - voltageAngle
	return math.Cos(angle * math.Pi / 180)
}

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < eps
}

func HasVDrop(
	voltageType coreenum.CTXEnumVoltageType,
	vDropVTM, vDropVTR, vDropITM, vDropITR float64,
	voltageL1, voltageL2, voltageL3 float64,
	currentL1, currentL2, currentL3 float64,
) bool {
	var upperVoltageLimit float64
	var lowerCurrentLimit float64

	switch voltageType {
	case coreenum.CTXEnumVoltageTypeTM:
		upperVoltageLimit = vDropVTM
		lowerCurrentLimit = vDropITM

	case coreenum.CTXEnumVoltageTypeTR:
		upperVoltageLimit = vDropVTR
		lowerCurrentLimit = vDropITR

	default:
		return false
	}

	return (voltageL1 < upperVoltageLimit && voltageL1 >= 0 && currentL1 > lowerCurrentLimit) ||
		(voltageL2 < upperVoltageLimit && voltageL2 >= 0 && currentL2 > lowerCurrentLimit) ||
		(voltageL3 < upperVoltageLimit && voltageL3 >= 0 && currentL3 > lowerCurrentLimit)
}

func HasVLoss(
	voltageType coreenum.CTXEnumVoltageType,
	vLossVTM, vLossVTR, vLossITM, vLossITR float64,
	phase int,
	voltageL1, voltageL2, voltageL3 float64,
	currentL1, currentL2, currentL3 float64,
) bool {

	if phase != 3 {
		return false
	}

	var targetVoltage, targetCurrent float64

	switch voltageType {
	case coreenum.CTXEnumVoltageTypeTM:
		targetVoltage = vLossVTM
		targetCurrent = vLossITM

	case coreenum.CTXEnumVoltageTypeTR:
		targetVoltage = vLossVTR
		targetCurrent = vLossITR

	default:
		return false
	}

	return (floatEqual(voltageL1, targetVoltage) && currentL1 > targetCurrent) ||
		(floatEqual(voltageL2, targetVoltage) && currentL2 > targetCurrent) ||
		(floatEqual(voltageL3, targetVoltage) && currentL3 > targetCurrent)
}

func HasCosPhiKecil(
	voltageType coreenum.CTXEnumVoltageType,
	measurementType coreenum.CTXEnumMeasurementType,
	cosPhiKecilUpperLimitTM, cosPhiKecilUpperLimitTR, cosPhiKecilITM, cosPhiKecilITR float64,
	powerFactorL1, powerFactorL2, powerFactorL3 float64,
	currentL1, currentL2, currentL3 float64,
) bool {
	if measurementType != coreenum.CTXEnumMeasurementTypeTakLangsung {
		return false
	}
	var (
		upperLimitPowerFactor float64
		lowerLimitI           float64
	)
	switch voltageType {
	case coreenum.CTXEnumVoltageTypeTM:
		upperLimitPowerFactor = cosPhiKecilUpperLimitTM
		lowerLimitI = cosPhiKecilITM
	case coreenum.CTXEnumVoltageTypeTR:
		upperLimitPowerFactor = cosPhiKecilUpperLimitTR
		lowerLimitI = cosPhiKecilITR
	default:
		return false
	}
	return (math.Abs(powerFactorL1) <= upperLimitPowerFactor && currentL1 > lowerLimitI) ||
		(math.Abs(powerFactorL2) <= upperLimitPowerFactor && currentL2 > lowerLimitI) ||
		(math.Abs(powerFactorL3) <= upperLimitPowerFactor && currentL3 > lowerLimitI)
}

func HasILoss(
	voltageType coreenum.CTXEnumVoltageType,
	iLossITM, iLossITR, iLossIMaxTM, iLossIMaxTR float64,
	currentL1, currentL2, currentL3, currentMax float64,
) bool {
	var upperLimit, lowerLimit float64

	switch voltageType {
	case coreenum.CTXEnumVoltageTypeTM:
		upperLimit = iLossITM
		lowerLimit = iLossIMaxTM
	case coreenum.CTXEnumVoltageTypeTR:
		upperLimit = iLossITR
		lowerLimit = iLossIMaxTR
	default:
		return false
	}

	return currentMax > lowerLimit && (currentL1 <= upperLimit || currentL2 <= upperLimit || currentL3 <= upperLimit)
}

func HasInGreaterIMax(
	voltageType coreenum.CTXEnumVoltageType,
	inGreaterIMaxInTM, inGreaterIMaxInTR float64,
	iN, iMax, iMin float64,
) bool {
	var neutralCurrentThreshold float64

	switch voltageType {
	case coreenum.CTXEnumVoltageTypeTM:
		neutralCurrentThreshold = inGreaterIMaxInTM
	case coreenum.CTXEnumVoltageTypeTR:
		neutralCurrentThreshold = inGreaterIMaxInTR
	default:
		return false
	}

	return iN > (iMax-iMin+5) && (iN > neutralCurrentThreshold)
}

var switchDirectCurrentThreshold = map[int64]float64{
	450:   2,
	900:   4,
	1300:  6,
	2200:  10,
	3500:  16,
	4400:  20,
	5500:  25,
	6600:  10,
	7700:  35,
	10600: 16,
	11000: 50,
	13200: 20,
	16500: 25,
	23000: 35,
	33000: 50,
	41500: 63,
}

const (
	overCurrentMultiplier = 1.4
	maxDirectPower        = 53000
	directPowerDivider    = 660.0
)

func HasOverCurrent(
	voltageType coreenum.CTXEnumVoltageType,
	overCurrentIMaxTM,
	overCurrentIMaxTR float64,
	power int64,
	currentMax float64,
) bool {

	if currentLimit, ok := switchDirectCurrentThreshold[power]; ok {
		return currentMax > currentLimit*overCurrentMultiplier
	}

	if power < maxDirectPower {
		return currentMax > (float64(power)/directPowerDivider)*overCurrentMultiplier
	}

	switch voltageType {
	case coreenum.CTXEnumVoltageTypeTM:
		return currentMax > overCurrentIMaxTM
	case coreenum.CTXEnumVoltageTypeTR:
		return currentMax > overCurrentIMaxTR
	default:
		return false
	}
}

func HasOverVoltage(
	voltageType coreenum.CTXEnumVoltageType,
	overVoltageVMaxTM, overVoltageVMaxTR float64,
	voltageMax float64,
) bool {
	var overVoltageTreshhold float64

	switch voltageType {
	case coreenum.CTXEnumVoltageTypeTM:
		overVoltageTreshhold = overVoltageVMaxTM
	case coreenum.CTXEnumVoltageTypeTR:
		overVoltageTreshhold = overVoltageVMaxTR
	default:
		return false
	}

	return voltageMax > overVoltageTreshhold
}

func HasReversePower(
	voltageType coreenum.CTXEnumVoltageType,
	reversePowerVTM, reversePowerVTR, reversePowerITM, reversePowerITR float64,
	activePowerL1, activePowerL2, activePowerL3, currentL1, currentL2, currentL3 float64,
) bool {
	var activePowerTreshhold float64
	var currentTreshhold float64
	switch voltageType {
	case coreenum.CTXEnumVoltageTypeTM:
		activePowerTreshhold = reversePowerVTM
		currentTreshhold = reversePowerITM
	case coreenum.CTXEnumVoltageTypeTR:
		activePowerTreshhold = reversePowerVTR
		currentTreshhold = reversePowerITR
	default:
		return false
	}

	return (activePowerL1 < activePowerTreshhold && currentL1 > currentTreshhold) ||
		(activePowerL2 < activePowerTreshhold && currentL2 > currentTreshhold) ||
		(activePowerL3 < activePowerTreshhold && currentL3 > currentTreshhold)
}

func Mean(numbers []float64) float64 {
	if len(numbers) == 0 {
		return 0.0
	}
	var (
		sum float64
	)
	for _, num := range numbers {
		sum += num
	}
	return sum / float64(len(numbers))
}

func HasUnbalanceCurrent(
	measurementType coreenum.CTXEnumMeasurementType,
	voltageType coreenum.CTXEnumVoltageType,
	iUnbalanceTolTM, iUnbalanceTolTR, iUnbalanceITM, iUnbalanceITR float64,
	currentL1, currentL2, currentL3 float64,
) bool {
	if measurementType != coreenum.CTXEnumMeasurementTypeTakLangsung {
		return false
	}
	var (
		currentUnbalanceTreshhold float64
		currentTreshhold          float64
	)
	switch voltageType {
	case coreenum.CTXEnumVoltageTypeTM:
		currentUnbalanceTreshhold = iUnbalanceTolTM
		currentTreshhold = iUnbalanceITM
	case coreenum.CTXEnumVoltageTypeTR:
		currentUnbalanceTreshhold = iUnbalanceTolTR
		currentTreshhold = iUnbalanceITR
	default:
		return false
	}
	currentAvg := Mean([]float64{currentL1, currentL2, currentL3})
	if floatEqual(currentAvg, 0) {
		return false
	}
	return (math.Abs(currentL1-currentAvg)/currentAvg >= currentUnbalanceTreshhold && currentL1 > currentTreshhold) ||
		(math.Abs(currentL2-currentAvg)/currentAvg >= currentUnbalanceTreshhold && currentL2 > currentTreshhold) ||
		(math.Abs(currentL3-currentAvg)/currentAvg >= currentUnbalanceTreshhold && currentL3 > currentTreshhold)
}

func HasActivePowerLost(
	billReffKwH int,
	pLossI float64,
	activePowerL1, activePowerL2, activePowerL3 float64,
	currentL1, currentL2, currentL3 float64,
) bool {
	if billReffKwH != 1 {
		return false
	}
	activePowerMax := math.Max(activePowerL1, math.Max(activePowerL2, activePowerL3))
	if activePowerMax <= 0 {
		return false
	}
	return activePowerMax > 0 &&
		(floatEqual(activePowerL1, 0) && currentL1 > pLossI ||
			floatEqual(activePowerL2, 0) && currentL2 > pLossI ||
			floatEqual(activePowerL3, 0) && currentL3 > pLossI)

}

func HasILowVLow(
	voltageType coreenum.CTXEnumVoltageType,
	iLowVLowTM, iLowVLowTR float64,
	currentL1, currentL2, currentL3 float64,
	voltageL1, voltageL2, voltageL3 float64,
) bool {
	var voltageDifference float64

	switch voltageType {
	case coreenum.CTXEnumVoltageTypeTM:
		voltageDifference = iLowVLowTM
	case coreenum.CTXEnumVoltageTypeTR:
		voltageDifference = iLowVLowTR
	default:
		return false
	}

	return (currentL1 < currentL2 && voltageL1 <= voltageL2 && math.Abs(voltageL1-voltageL2) >= voltageDifference) ||
		(currentL1 < currentL3 && voltageL1 <= voltageL3 && math.Abs(voltageL1-voltageL3) >= voltageDifference) ||
		(currentL2 < currentL3 && voltageL2 <= voltageL3 && math.Abs(voltageL2-voltageL3) >= voltageDifference)
}

func HasFreeze(
	voltageMax float64,
) bool {
	return floatEqual(voltageMax, 0)
}

func HasCurrentLoop(
	currentL1, currentL2, currentL3 float64,
	currentAngleL1, currentAngleL2, currentAngleL3 float64,
	power int64,
) bool {
	if power < 53000 {
		return false
	}
	return (math.Abs(currentL1-currentL2) < 0.02 && currentL1+currentL2 > 0.6 && math.Abs(180-math.Abs(currentAngleL1-currentAngleL2)) < 1) ||
		(math.Abs(currentL1-currentL3) < 0.02 && currentL1+currentL3 > 0.6 && math.Abs(180-math.Abs(currentAngleL1-currentAngleL3)) < 1) ||
		(math.Abs(currentL2-currentL3) < 0.02 && currentL2+currentL3 > 0.6 && math.Abs(180-math.Abs(currentAngleL2-currentAngleL3)) < 1)
}
