package gontentful

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/lib/pq"
	"github.com/mitchellh/mapstructure"
)

type rowField struct {
	fieldName  string
	fieldValue interface{}
}

type columnData struct {
	fieldColumns     []string
	columnReferences map[string]string
}

func appendTables(tablesByName map[string]*PGSyncTable, conTablesByName map[string]*PGSyncConTable, item *Entry, tableName string, fieldColumns []string, refColumns map[string]string, templateFormat bool) {
	fieldsByLocale := make(map[string][]*rowField, 0)

	// iterate over fields
	for fieldName, f := range item.Fields {
		locFields, ok := f.(map[string]interface{})
		if !ok {
			continue // no locale, continue
		}

		// snace_case column name
		columnName := toSnakeCase(fieldName)

		// iterate over locale fields
		for locale, fieldValue := range locFields {
			// create table
			tbl := tablesByName[tableName]
			if tbl == nil {
				tbl = newPGSyncTable(tableName, fieldColumns)
				tablesByName[tableName] = tbl
			}

			// collect row fields by locale
			fieldsByLocale[locale] = append(fieldsByLocale[locale], &rowField{columnName, fieldValue})
		}
	}

	// append rows with fields to tables
	for locale, rowFields := range fieldsByLocale {
		// table
		tbl := tablesByName[tableName]
		if tbl != nil {
			appendRowsToTable(item, tbl, rowFields, fieldColumns, templateFormat, conTablesByName, refColumns, tableName, locale)
		}
	}
}

func appendRowsToTable(item *Entry, tbl *PGSyncTable, rowFields []*rowField, fieldColumns []string, templateFormat bool, conTables map[string]*PGSyncConTable, refColumns map[string]string, tableName string, locale string) {
	fieldValues := make(map[string]interface{}, len(fieldColumns))
	for _, rowField := range rowFields {
		fieldValues[rowField.fieldName] = convertFieldValue(rowField.fieldValue, templateFormat)
		assetFile, ok := fieldValues[rowField.fieldName].(*AssetFile)
		if ok {
			url := assetFile.URL
			fileName := assetFile.FileName
			contentType := assetFile.ContentType
			if templateFormat {
				url = fmt.Sprintf("'%s'", url)
				fileName = fmt.Sprintf("'%s'", fileName)
				contentType = fmt.Sprintf("'%s'", contentType)
			}
			fieldValues["url"] = url
			fieldValues["file_name"] = fileName
			fieldValues["content_type"] = contentType
		}
		// append con tables with Array Links
		if refColumns[rowField.fieldName] != "" {
			links, ok := rowField.fieldValue.([]interface{})
			if ok {
				conRows := make([][]interface{}, 0)
				sysID := item.Sys.ID
				for _, e := range links {
					f, ok := e.(map[string]interface{})
					if ok {
						id := convertSys(f, templateFormat)
						if id != "" {
							row := []interface{}{sysID, id, locale}
							conRows = append(conRows, row)
						}
					}
				}
				conTableName := getConTableName(tableName, refColumns[rowField.fieldName])
				conTables[conTableName] = &PGSyncConTable{
					TableName: conTableName,
					Columns:   []string{tableName, refColumns[rowField.fieldName], "_locale"},
					Rows:      conRows,
				}
			}
		}
	}
	row := newPGSyncRow(item, fieldColumns, fieldValues, locale)
	tbl.Rows = append(tbl.Rows, row)
}

func convertFieldValue(v interface{}, t bool) interface{} {
	switch f := v.(type) {

	case map[string]interface{}:
		if f["sys"] != nil {
			s := convertSys(f, t)
			if s != "" {
				return s
			}
		} else if f["fileName"] != nil {
			var v *AssetFile
			mapstructure.Decode(f, &v)
			return v
		} else {
			data, err := json.Marshal(f)
			if err != nil {
				log.Fatal("failed to marshal content field")
			}
			return string(data)
		}

	case []interface{}:
		arr := make([]string, 0)
		for i := 0; i < len(f); i++ {
			fs := convertFieldValue(f[i], t)
			arr = append(arr, fmt.Sprintf("%v", fs))
		}
		if t {
			return fmt.Sprintf("'{%s}'", strings.ReplaceAll(strings.Join(arr, ","), "'", "\""))
		}
		return pq.Array(arr)

	case []string:
		arr := make([]string, 0)
		for i := 0; i < len(f); i++ {
			fs := convertFieldValue(f[i], t)
			arr = append(arr, fmt.Sprintf("%v", fs))
		}
		if t {
			return fmt.Sprintf("'{%s}'", strings.ReplaceAll(strings.Join(arr, ","), "'", "\""))
		}
		return pq.Array(arr)
	case string:
		if t {
			return fmt.Sprintf("'%s'", strings.ReplaceAll(v.(string), "'", "''"))
		}
	}

	return v
}

func convertSys(f map[string]interface{}, t bool) string {
	s, ok := f["sys"].(map[string]interface{})
	if ok {
		if s["type"] == "Link" {
			if t {
				return fmt.Sprintf("'%v'", s["id"])
			}
			return fmt.Sprintf("%v", s["id"])
		}
	}
	return ""
}

func getColumnsByContentType(types []*ContentType) map[string]*columnData {
	typeColumns := make(map[string]*columnData)
	for _, t := range types {
		if typeColumns[t.Sys.ID] == nil {
			fieldColumns := make([]string, 0)
			refColumns := make(map[string]string)
			for _, f := range t.Fields {
				if !f.Omitted {
					colName := toSnakeCase(f.ID)
					fieldColumns = append(fieldColumns, colName)
					if f.Items != nil {
						linkType := getFieldLinkType(f.Items.LinkType, f.Items.Validations)
						if linkType != "" {
							refColumns[colName] = linkType
						}
					}
				}
			}
			typeColumns[t.Sys.ID] = &columnData{fieldColumns, refColumns}
		}
	}
	return typeColumns
}
