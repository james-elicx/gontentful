package gontentful

import (
	"bytes"
	"database/sql"
	"fmt"
	"text/template"

	"github.com/lib/pq"
)

type PGSyncRow struct {
	SysID            string
	FieldColumns     []string
	FieldValues      map[string]interface{}
	MetaColumns      []string
	Version          int
	PublishedVersion int
	CreatedAt        string
	UpdatedAt        string
	PublishedAt      string
}

type PGSyncTable struct {
	TableName string
	Columns   []string
	Rows      []*PGSyncRow
}

type PGSyncSchema struct {
	SchemaName string
	Tables     []*PGSyncTable
	Deleted    []string
}

func NewPGSyncSchema(schemaName string, types []*ContentType, items []*Entry) *PGSyncSchema {
	schema := &PGSyncSchema{
		SchemaName: schemaName,
		Tables:     make([]*PGSyncTable, 0),
		Deleted:    make([]string, 0),
	}

	for _, item := range items {
		switch item.Sys.Type {
		case ENTRY:
			contentType := item.Sys.ContentType.Sys.ID
			fieldColumns := getFieldColumns(types, contentType)
			entryTables := makeTables(item, contentType, fieldColumns)
			schema.Tables = append(schema.Tables, entryTables...)
			break
		case ASSET:
			fieldColumns := []string{"title", "url", "filename", "contenttype"}
			assetTables := makeTables(item, "_assets", fieldColumns)
			schema.Tables = append(schema.Tables, assetTables...)
			break
		case DELETED_ENTRY, DELETED_ASSET:
			schema.Deleted = append(schema.Deleted, item.Sys.ID)
			break
		}
	}

	// create a "global" entries table to store all entries with sysid for later delete
	tableNamesBySysID := make(map[string]string, 0)
	for _, table := range schema.Tables {
		for _, row := range table.Rows {
			tableNamesBySysID[row.SysID] = table.TableName
		}
	}
	entriesTable := newPGSyncTable("_entries", []string{"tablename"}, []string{})
	for sysid, tablename := range tableNamesBySysID {
		enrtiesRow := &PGSyncRow{
			SysID:        sysid,
			FieldColumns: []string{"tablename"},
			FieldValues: map[string]interface{}{
				"tablename": tablename,
			},
		}
		entriesTable.Rows = append(entriesTable.Rows, enrtiesRow)
	}
	schema.Tables = append(schema.Tables, entriesTable)

	return schema
}

func newPGSyncTable(tableName string, fieldColumns []string, metaColumns []string) *PGSyncTable {
	columns := []string{"sysid"}
	columns = append(columns, fieldColumns...)
	columns = append(columns, metaColumns...)

	return &PGSyncTable{
		TableName: tableName,
		Columns:   columns,
		Rows:      make([]*PGSyncRow, 0),
	}
}

func newPGSyncRow(item *Entry, fieldColumns []string, fieldValues map[string]interface{}, metaColumns []string) *PGSyncRow {
	row := &PGSyncRow{
		SysID:            item.Sys.ID,
		FieldColumns:     fieldColumns,
		FieldValues:      fieldValues,
		MetaColumns:      metaColumns,
		Version:          item.Sys.Version,
		CreatedAt:        item.Sys.CreatedAt,
		UpdatedAt:        item.Sys.UpdatedAt,
		PublishedVersion: item.Sys.PublishedVersion,
		PublishedAt:      item.Sys.PublishedAt,
	}
	if row.Version == 0 {
		row.Version = item.Sys.Revision
	}
	if row.PublishedVersion == 0 {
		row.PublishedVersion = row.Version
	}
	if len(row.UpdatedAt) == 0 {
		row.UpdatedAt = row.CreatedAt
	}
	if len(row.PublishedAt) == 0 {
		row.PublishedAt = row.UpdatedAt
	}
	return row
}

func (r *PGSyncRow) Fields() []interface{} {
	values := []interface{}{
		r.SysID,
	}
	for _, fieldColumn := range r.FieldColumns {
		values = append(values, r.FieldValues[fieldColumn])
	}
	for _, metaColumn := range r.MetaColumns {
		switch metaColumn {
		case "version":
			values = append(values, r.Version)
		case "created_at":
			values = append(values, r.CreatedAt)
		case "created_by":
			values = append(values, "sync")
		case "updated_at":
			values = append(values, r.UpdatedAt)
		case "updated_by":
			values = append(values, "sync")
		case "published_at":
			values = append(values, r.PublishedAt)
		case "published_by":
			values = append(values, "sync")
		}
	}
	return values
}

func (s *PGSyncSchema) Exec(databaseURL string, initSync bool) error {
	db, _ := sql.Open("postgres", databaseURL)

	_, err := db.Exec(fmt.Sprintf("set search_path='%s'", s.SchemaName))
	if err != nil {
		return err
	}

	// init sync
	if initSync {
		return s.bulkInsert(db)
	}

	// insert and/or delete changes
	return s.deltaSync(db)
}

func (s *PGSyncSchema) bulkInsert(db *sql.DB) error {
	txn, err := db.Begin()
	if err != nil {
		return err
	}

	for _, tbl := range s.Tables {
		if len(tbl.Rows) == 0 {
			continue
		}

		stmt, err := txn.Prepare(pq.CopyIn(tbl.TableName, tbl.Columns...))
		if err != nil {
			return err
		}

		for _, row := range tbl.Rows {
			_, err = stmt.Exec(row.Fields()...)
			if err != nil {
				return err
			}
		}

		_, err = stmt.Exec()
		if err != nil {
			return err
		}

		err = stmt.Close()
		if err != nil {
			return err
		}
	}

	return txn.Commit()
}

func (s *PGSyncSchema) deltaSync(db *sql.DB) error {
	tmpl, err := template.New("").Parse(pgSyncTemplate)
	if err != nil {
		return err
	}

	var buff bytes.Buffer
	err = tmpl.Execute(&buff, s)
	if err != nil {
		return err
	}

	txn, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = txn.Exec(buff.String())
	if err != nil {
		return err
	}

	return txn.Commit()
}
