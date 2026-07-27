package helpergenerator

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/hasifpri/dancok"
	"logisfy/core"
	"strconv"
	"strings"
)

func GenerateQueryInfoPostgreSQL(fiberCtx *fiber.Ctx) (core.QueryInfo, error) {
	filter := fiberCtx.Query("filter")
	sort := fiberCtx.Query("sort")
	page := fiberCtx.Query("page")
	pageSize := fiberCtx.Query("pageSize")

	return ParseQueryInfoPostgreSQL(filter, sort, page, pageSize)
}
func ParseQueryInfoPostgreSQL(filter, sort, page, pageSize string) (core.QueryInfo, error) {
	if page == "" {
		page = "1"
	}
	if pageSize == "" {
		pageSize = "20"
	}

	queryInfo := core.QueryInfo{}
	selectParameter := dancok.SelectParameter{}

	pageDescriptor := dancok.PageDescriptor{}
	pi, _ := strconv.ParseInt(page, 10, 32)
	pageDescriptor.PageIndex = int32(pi)
	ps, _ := strconv.ParseInt(pageSize, 10, 32)
	pageDescriptor.PageSize = int32(ps)
	selectParameter.PageDescriptor = pageDescriptor

	if sort != "" {
		sortList := strings.Split(sort, "|")

		for _, sortCriteria := range sortList {
			sortInfo := strings.Split(sortCriteria, ":")

			if len(sortInfo) != 2 {
				return core.QueryInfo{}, fmt.Errorf("sort pattern invalid")
			}

			sortDescriptor := dancok.SortDescriptor{}
			sortDescriptor.FieldName = sortInfo[0]
			if sortInfo[1] == "asc" {
				sortDescriptor.SortDirection = dancok.Ascending
			} else {
				sortDescriptor.SortDirection = dancok.Descending
			}

			selectParameter.SortDescriptors = append(selectParameter.SortDescriptors, sortDescriptor)
		}
	}

	filterList := strings.Split(filter, "|")
	for _, filterCriteria := range filterList {
		if strings.Contains(filterCriteria, ":") {
			filterInfo := strings.Split(filterCriteria, ":")

			filterDescriptor := dancok.FilterDescriptor{}
			filterDescriptor.FieldName = filterInfo[0]

			// to can filter by id
			if strings.Contains(filterInfo[1], ",") {
				values := strings.Split(filterInfo[1], ",")
				rangeValues := make([]any, len(values))
				for i, v := range values {
					rangeValues[i] = v
				}
				filterDescriptor.RangeValues = rangeValues
			} else {
				filterDescriptor.Value = filterInfo[1]
			}

			switch opt := filterInfo[2]; opt {
			case "equals":
				filterDescriptor.Operator = dancok.IsEqual
			case "notequals":
				filterDescriptor.Operator = dancok.IsNotEqual
			case "greaterthan":
				filterDescriptor.Operator = dancok.IsMoreThan
			case "greaterthanorequal":
				filterDescriptor.Operator = dancok.IsMoreThanOrEqual
			case "lessthan":
				filterDescriptor.Operator = dancok.IsLessThan
			case "lessthanorequal":
				filterDescriptor.Operator = dancok.IsLessThanOrEqual
			case "contains":
				filterDescriptor.Operator = dancok.IsContain
			case "startswith":
				filterDescriptor.Operator = dancok.IsBeginWith
			case "endswith":
				filterDescriptor.Operator = dancok.IsEndWith
			case "isin":
				if !strings.Contains(filterInfo[1], ",") {
					rangeValues := make([]any, 1)
					rangeValues[0] = filterInfo[1]
					filterDescriptor.RangeValues = rangeValues
				}
				filterDescriptor.Operator = dancok.IsIn
			default:
				filterDescriptor.Operator = dancok.IsEqual
			}

			selectParameter.FilterDescriptors = append(selectParameter.FilterDescriptors, filterDescriptor)
		}
	}
	queryInfo.SelectParameter = selectParameter
	queryInfo.Filter = filter
	queryInfo.Sort = sort

	return queryInfo, nil
}
