package core

import "github.com/hasifpri/dancok"

type QueryInfo struct {
	Filter          string
	Sort            string
	SelectParameter dancok.SelectParameter
}
