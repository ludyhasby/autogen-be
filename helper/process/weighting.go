package helperprocess

func VDropWeighted(vDrop bool, power int64, VIndirectDrop, VDirectDrop float64) float64 {
	if !vDrop {
		return 0
	}
	if power >= 53000 {
		return VIndirectDrop
	}
	return VDirectDrop
}

func CommonWeighted(v bool, w float64) float64 {
	if !v {
		return 0
	}
	return w
}
