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

func appendTables(tablesByName map[string]*PGSyncTable, item *Entry, baseName string, fieldColumns []string, templateFormat bool) {
	fieldsByLocale := make(map[string][]*rowField, 0)

	tblMetaColumns := []string{"version", "created_at", "created_by", "updated_at", "updated_by"}
	pubMetaColumns := []string{"version", "published_at", "published_by"}

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
			tableName := fmtTableName(baseName, locale)
			tbl := tablesByName[tableName]
			if tbl == nil {
				tbl = newPGSyncTable(tableName, fieldColumns, tblMetaColumns)
				tablesByName[tableName] = tbl
			}

			// create publish table
			pubTableName := fmtTablePublishName(baseName, locale)
			pubTable := tablesByName[pubTableName]
			if pubTable == nil {
				pubTable := newPGSyncTable(pubTableName, fieldColumns, pubMetaColumns)
				tablesByName[pubTableName] = pubTable
			}

			// collect row fields by locale
			fieldsByLocale[locale] = append(fieldsByLocale[locale], &rowField{columnName, fieldValue})
		}
	}

	// append rows with fields to tables
	for locale, rowFields := range fieldsByLocale {
		// table
		tableName := fmtTableName(baseName, locale)
		tbl := tablesByName[tableName]
		if tbl != nil {
			appendRowsToTable(item, tbl, rowFields, fieldColumns, tblMetaColumns, templateFormat)
		}

		// publish table
		pubTableName := fmtTablePublishName(baseName, locale)
		pubTable := tablesByName[pubTableName]
		if pubTable != nil {
			appendRowsToTable(item, pubTable, rowFields, fieldColumns, pubMetaColumns, templateFormat)
		}
	}
}

func appendRowsToTable(item *Entry, tbl *PGSyncTable, rowFields []*rowField, fieldColumns []string, metaColums []string, templateFormat bool) {
	fieldValues := make(map[string]interface{}, len(fieldColumns))
	for _, rowField := range rowFields {
		fieldValues[rowField.fieldName] = convertFieldValue(rowField.fieldValue, templateFormat)
		assetFile, ok := fieldValues[rowField.fieldName].(*AssetFile)
		if ok {
			fieldValues["url"] = assetFile.URL
			fieldValues["file_name"] = assetFile.FileName
			fieldValues["content_type"] = assetFile.ContentType
		}
	}
	row := newPGSyncRow(item, fieldColumns, fieldValues, metaColums)
	tbl.Rows = append(tbl.Rows, row)
}

func convertFieldValue(v interface{}, t bool) interface{} {
	switch f := v.(type) {

	case map[string]interface{}:
		if f["sys"] != nil {
			s, ok := f["sys"].(map[string]interface{})
			if ok {
				if s["type"] == "Link" {
					if t {
						return fmt.Sprintf("'%v'", s["id"])
					}
					return fmt.Sprintf("%v", s["id"])
				}
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
			return fmt.Sprintf("'%v'", v)
		}
	}

	return v
}
