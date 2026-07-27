package helperchecker

import (
	"log/slog"
	helperexception "logisfy/helper/exception"
)

func AssertFoundData[T interface{ PK() int64 }](log *slog.Logger, caller, where, notFoundMsg string, e T) *helperexception.Exception {
	if e.PK() == 0 {
		log.Info(caller, where, "Error", notFoundMsg)
		return helperexception.NotFound(notFoundMsg)
	}
	return nil
}

func AssertAccountUUIDMatch[T interface{ AccountUUID() string }](
	log *slog.Logger,
	caller, where, notFoundMsg string,
	expectedTenant string,
	e T,
) *helperexception.Exception {
	if e.AccountUUID() != expectedTenant {
		log.Info(caller, where, "Error", notFoundMsg)
		return helperexception.NotFound(notFoundMsg)
	}
	return nil
}
