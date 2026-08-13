package helperchecker

import (
	"context"
	"logisfy/core"
	coreenum "logisfy/core/enum"
	"strconv"

	"github.com/hasifpri/dancok"
)

func FilterUserID(ctx context.Context, param *core.QueryInfo) {
	val := ctx.Value(string(coreenum.CTXEnumIDUserID))
	if val == nil {
		return
	}
	userIDStr, ok := val.(string)
	if !ok {
		return
	}
	_, err := strconv.Atoi(userIDStr)
	if err != nil {
		return
	}
	param.SelectParameter.FilterDescriptors = append(param.SelectParameter.FilterDescriptors, dancok.FilterDescriptor{
		FieldName: "user_id",
		Value:     userIDStr,
		Operator:  dancok.IsEqual,
		Condition: dancok.And,
	})
}
