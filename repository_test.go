package gt

import (
	"gorm.io/gorm"
	"testing"
)

func TestRepository_Generate(t *testing.T) {
	type fields struct {
		Path        string
		DSN         string
		TableFields map[string][]string
		Prefix      string
		IsCreate    bool
		IsUpdate    bool
		dbInfo      DbInfo
		db          *gorm.DB
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "生成数据",
			fields: fields{
				Path: "/Users/Apple/Application/github/gt/repository",
				DSN:  "root:abc123@tcp(192.168.150.59:3306)/mohe?parseTime=true&loc=Local",
				TableFields: map[string][]string{
					"daily_sku_summary": {},
				},
				IsCreate: true,
				IsUpdate: true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Repository{
				Path:        tt.fields.Path,
				DSN:         tt.fields.DSN,
				TableFields: tt.fields.TableFields,
				Prefix:      tt.fields.Prefix,
				IsCreate:    tt.fields.IsCreate,
				IsUpdate:    tt.fields.IsUpdate,
				dbInfo:      tt.fields.dbInfo,
				db:          tt.fields.db,
			}
			if err := r.Generate(); (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
