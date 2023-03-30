package gontentful

import (
	"strings"
	"time"

	"github.com/moonwalker/moonbase/pkg/content"
)

func TransformModel(model *ContentType) (*content.Schema, error) {
	createdAt, _ := time.Parse(time.RFC3339Nano, model.Sys.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339Nano, model.Sys.UpdatedAt)
	schema := &content.Schema{
		ID:          model.Sys.ID,
		Name:        model.Name,
		Description: model.Description,
		CreatedAt:   &createdAt,
		CreatedBy:   "admin@moonwalker.tech",
		UpdatedAt:   &updatedAt,
		UpdatedBy:   "admin@moonwalker.tech",
		Version:     model.Sys.Version,
	}

	for _, item := range model.Fields {
		cf := &content.Field{
			ID:        item.ID,
			Label:     item.Name,
			Localized: item.Localized,
			Disabled:  item.Disabled,
		}

		if item.DefaultValue != nil {
			for _, dv := range item.DefaultValue {
				cf.DefaultValue = dv
				break
			}
		}

		if item.Required {
			cf.Validations = append(cf.Validations, &content.Validation{
				Type:  "required",
				Value: true,
			})
		}

		transformField(cf, item.Type, item.LinkType, item.Validations, item.Items)

		schema.Fields = append(schema.Fields, cf)
	}

	return schema, nil
}

func transformField(cf *content.Field, fieldType string, linkType string, validations []*FieldValidation, items *FieldTypeArrayItem) {
	for _, v := range validations {
		// if v.Unique {
		cf.Validations = append(cf.Validations, &content.Validation{
			Type:  "unique",
			Value: v.Unique,
		})
		// }
		if v.In != nil {
			cf.Validations = append(cf.Validations, &content.Validation{
				Type:  "in",
				Value: v.In,
			})
		}
		if v.Size != nil {
			cf.Validations = append(cf.Validations, &content.Validation{
				Type:  "size",
				Value: *v.Size,
			})
		}
		if v.Range != nil {
			cf.Validations = append(cf.Validations, &content.Validation{
				Type:  "range",
				Value: *v.Range,
			})
		}
		if v.Regexp != nil {
			cf.Validations = append(cf.Validations, &content.Validation{
				Type:  "regexp",
				Value: *v.Regexp,
			})
		}
	}

	switch fieldType {
	case "Symbol":
		cf.Type = "text"
		break
	case "Boolean":
		cf.Type = "bool"
		break
	case "Integer":
		cf.Type = "int"
		break
	case "Number":
		cf.Type = "float"
		break
	case "Text":
		cf.Type = "longtext"
		break
	case "Link":
		cf.Reference = true
		if linkType == "Asset" {
			cf.Type = "_asset"
		} else {
			cf.Type = getFieldLinkContentType(validations)
		}
		break
	case "Array":
		cf.List = true
		transformField(cf, items.Type, items.LinkType, items.Validations, nil)
		break
	case "Object":
		cf.Type = "json"
		break
	}
}

func transformValidationFields(vals []*content.Validation) ([]*FieldValidation, bool) {
	cfValidations := make([]*FieldValidation, 0)
	required := false
	for _, v := range vals {
		if v.Type == "unique" && v.Value == true {
			cfValidations = append(cfValidations, &FieldValidation{
				Unique: true,
			})
		}

		if v.Type == "required" && v.Value == true {
			required = true
		}

		if v.Type == "in" {
			strarr := make([]string, 0)
			for _, i := range v.Value.([]interface{}) {
				strarr = append(strarr, i.(string))
			}
			cfValidations = append(cfValidations, &FieldValidation{
				In: strarr,
			})
		}

		if v.Type == "size" {
			m, _ := v.Value.(map[string]interface{})
			rv := &RangeValidation{}
			for k, v := range m {
				if k == "min" {
					i := int(v.(float64))
					rv.Min = &i
				}
				if k == "max" {
					i := int(v.(float64))
					rv.Max = &i
				}
			}
			cfValidations = append(cfValidations, &FieldValidation{
				Size: rv,
			})
		}

		if v.Type == "regexp" {
			m, _ := v.Value.(map[string]interface{})
			rv := &RegexpValidation{}
			for k, v := range m {
				if k == "pattern" {
					rv.Pattern = int(v.(float64))
				}
				if k == "flags" {
					rv.Flags = int(v.(float64))
				}
			}
			cfValidations = append(cfValidations, &FieldValidation{
				Regexp: rv,
			})
		}
	}
	return cfValidations, required
}

func transformToContentfulField(cf *ContentTypeField, fieldType string, validations []*content.Validation, list bool, reference bool) {

	cfVals, required := transformValidationFields(validations)
	if required {
		cf.Required = true
	}
	if list {
		cf.Type = "Array"
		cf.Items = &FieldTypeArrayItem{}

		if reference {
			cf.Items.Type = "Link"
			cf.Items.LinkType = "Entry"
			if fieldType == "_asset" {
				cf.Items.LinkType = "Asset"
			}
			cf.Items.Validations = append(cfVals, &FieldValidation{
				LinkContentType: append(make([]string, 0), fieldType),
			})
		} else {
			cf.Items.Type = GetContentfulType(fieldType)
		}
	} else if reference {
		cf.Type = "Link"
		cf.LinkType = "Entry"
		if fieldType == "_asset" {
			cf.LinkType = "Asset"
		}
		cf.Validations = append(cfVals, &FieldValidation{
			LinkContentType: append(make([]string, 0), fieldType),
		})
	} else {
		cf.Type = GetContentfulType(fieldType)
		cf.Validations = cfVals
	}
}

func GetContentfulType(fieldType string) string {
	returnVal := ""
	switch fieldType {
	case "text":
		returnVal = "Symbol"
		break
	case "bool":
		returnVal = "Boolean"
		break
	case "int":
		returnVal = "Integer"
		break
	case "float":
		returnVal = "Number"
		break
	case "longtext":
		returnVal = "Text"
		break
	case "_asset":
		returnVal = "Asset"
	}
	return returnVal
}

func FormatSchema(schema *content.Schema) (*ContentType, error) {
	ct := &ContentType{
		Name:         schema.Name,
		Description:  schema.Description,
		DisplayField: schema.Fields[0].ID,
		Sys: &Sys{
			ID:      schema.ID,
			Version: schema.Version,
		},
	}
	if schema.CreatedAt != nil {
		ct.Sys.CreatedAt = schema.CreatedAt.Format(time.RFC3339Nano)
	}
	if schema.UpdatedAt != nil {
		ct.Sys.UpdatedAt = schema.UpdatedAt.Format(time.RFC3339Nano)
	}

	ct.Sys.CreatedBy = &Entry{
		Sys: &Sys{
			ID: schema.CreatedBy,
		},
	}
	ct.Sys.UpdatedBy = &Entry{
		Sys: &Sys{
			ID: schema.CreatedBy,
		},
	}

	for _, f := range schema.Fields {
		ctf := &ContentTypeField{
			ID:          f.ID,
			Name:        f.Label,
			Localized:   f.Localized,
			Disabled:    f.Disabled,
			Required:    false,
			Omitted:     false,
			Validations: make([]*FieldValidation, 0),
		}
		if f.DefaultValue != nil {
			ctf.DefaultValue = make(map[string]interface{})
			ctf.DefaultValue["en"] = f.DefaultValue
		}
		transformToContentfulField(ctf, f.Type, f.Validations, f.List, f.Reference)

		ct.Fields = append(ct.Fields, ctf)
	}

	return ct, nil
}

func TransformEntry(locales *Locales, model *Entry) (map[string]*content.ContentData, error) {
	res := make(map[string]*content.ContentData, 0)
	for _, loc := range locales.Items {
		data := &content.ContentData{
			ID:     model.Sys.ID,
			Fields: make(map[string]interface{}),
		}

		for fn, fv := range model.Fields {
			locValues, ok := fv.(map[string]interface{})
			if !ok {
				continue // no locale value, continue
			}

			locValue := locValues[strings.ToLower(loc.Code)]
			if locValue == nil {
				locValue = locValues[defaultLocale]
			}

			if lsysl, ok := locValue.([]interface{}); ok {
				for _, lsyso := range lsysl {
					if lsys, ok := lsyso.(map[string]interface{}); ok {
						sid := getSysID(lsys)
						if sid == nil {
							break
						}
						if data.Fields[fn] == nil {
							data.Fields[fn] = make([]interface{}, 0)
						}
						data.Fields[fn] = append(data.Fields[fn].([]interface{}), sid)
					}
				}
			} else {
				if lsys, ok := locValue.(map[string]interface{}); ok {
					data.Fields[fn] = getSysID(lsys)
				}
			}

			if data.Fields[fn] == nil {
				data.Fields[fn] = locValue
			}
		}

		data.CreatedAt = getSysDate(model.Sys.CreatedAt)
		data.CreatedBy = "admin"
		data.UpdatedAt = getSysDate(model.Sys.UpdatedAt)
		data.UpdatedBy = "admin"
		data.Version = model.Sys.Version
		res[strings.ToLower(loc.Code)] = data
	}

	return res, nil
}

func getSysDate(date string) *time.Time {
	var t time.Time
	t, _ = time.Parse(time.RFC3339Nano, date)
	return &t
}

func getSysID(lsys map[string]interface{}) interface{} {
	if sys, ok := lsys["sys"].(map[string]interface{}); ok {
		if sid, ok := sys["id"].(string); ok {
			return sid
		}
	}
	return nil
}

func FormatData(contentType string, id string, schemas map[string]*content.Schema, locData map[string]map[string]map[string]content.ContentData) (*Entry, map[string]string, error) {
	schema := schemas[contentType]
	contents := locData[contentType][id]

	refFields := make(map[string]*content.Field, 0)
	for _, sf := range schema.Fields {
		if sf.Reference {
			refFields[sf.ID] = sf
		}
	}

	entry, includes, err := formatEntry(id, contentType, contents, refFields)
	if err != nil {
		return nil, nil, err
	}

	return entry, includes, nil
}

func formatEntry(id string, contentType string, contents map[string]content.ContentData, refFields map[string]*content.Field) (*Entry, map[string]string, error) {
	includes := make(map[string]string)

	sysType := "Entry"
	if contentType == "_asset" {
		sysType = "Asset"
	}

	e := &Entry{
		Sys: &Sys{
			ID:      id,
			Type:    sysType,
			Version: contents[defaultLocale].Version,
			ContentType: &ContentType{
				Sys: &Sys{
					Type:     "Link",
					LinkType: "ContentType",
					ID:       contentType,
				},
			},
		},
		Fields: make(map[string]interface{}),
	}

	if contents[defaultLocale].CreatedAt != nil {
		e.Sys.CreatedAt = contents[defaultLocale].CreatedAt.Format(time.RFC3339Nano)
	}
	if contents[defaultLocale].UpdatedAt != nil {
		e.Sys.UpdatedAt = contents[defaultLocale].UpdatedAt.Format(time.RFC3339Nano)
	}

	for loc, data := range contents {
		for fn, fv := range data.Fields {
			if fv == nil {
				continue
			}
			if e.Fields[fn] == nil {
				e.Fields[fn] = make(map[string]interface{})
			}

			if rf := refFields[fn]; rf != nil {
				if rf.List {
					if rl, ok := fv.([]interface{}); ok {
						refList := make([]string, 0)
						for _, r := range rl {
							if rid, ok := r.(string); ok {
								refList = append(refList, rid)
								includes[rid] = rf.Type
							}
						}
						e.Fields[fn].(map[string]interface{})[loc] = refList
					}
				} else {
					e.Fields[fn].(map[string]interface{})[loc] = fv.(string)
					includes[fv.(string)] = rf.Type
				}
			} else {
				e.Fields[fn].(map[string]interface{})[loc] = fv
			}
		}
	}

	return e, includes, nil
}
