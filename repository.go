package gt

import (
	"fmt"
	"github.com/duke-git/lancet/fileutil"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gt/keywords"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type KeywordsExpress struct {
	Keywords string
	Express  string
}

var msKeywordsToGoKeywordsMap = map[string]KeywordsExpress{
	"int": {
		Keywords: "int",
		Express:  ">0",
	},
	"tinyint": {
		Keywords: "int",
		Express:  ">0",
	},
	"smallint": {
		Keywords: "int",
		Express:  ">0",
	},
	"mediumint": {
		Keywords: "int",
		Express:  ">0",
	},
	"bigint": {
		Keywords: "int",
		Express:  ">0",
	},
	"varchar": {
		Keywords: "int",
		Express:  `!=""`,
	},
	"char": {
		Keywords: "int",
		Express:  `!=""`,
	},
	"text": {
		Keywords: "int",
		Express:  `!=""`,
	},
	"date": {
		Keywords: "time.Time",
		Express:  ``,
	},
	"datetime": {
		Keywords: "time.Time",
		Express:  ``,
	},
	"timestamp": {
		Keywords: "time.Time",
		Express:  ``,
	},
	"float": {
		Keywords: "int",
		Express:  ">0",
	},
	"double": {
		Keywords: "int",
		Express:  ">0",
	},
	"json": {
		Keywords: "int",
		Express:  `!=""`,
	},
}

type Repository struct {
	Path        string // 生成代码的路径
	DSN         string
	TableFields map[string][]string
	Prefix      string
	IsCreate    bool
	IsUpdate    bool
	dbInfo      DbInfo
	db          *gorm.DB // 数据库连接对象
}

type DbInfo struct {
	database string // 数据库名称
}

type TableColumns struct {
	Field   string `json:"field"`
	Type    string `json:"type"`
	Null    string `json:"null"`
	Key     string `json:"key"`
	Default string `json:"default"`
	Extra   string `json:"extra"`
	Comment string `json:"comment"`
}

func NewRepository(
	dsn string,
	tableFields map[string][]string,
	prefix string,
	path string,
) *Repository {
	return &Repository{
		DSN:         dsn,
		TableFields: tableFields,
		Prefix:      prefix,
		Path:        path,
	}
}

func (r *Repository) dsn() string {
	return r.DSN
}

func (r *Repository) connDb() error {
	if r.db != nil {
		return nil
	}
	db, err := gorm.Open(mysql.Open(r.dsn()), &gorm.Config{})
	if err != nil {
		panic(errors.Wrap(err, "failed to connect database"))
	}
	r.db = db
	return nil
}

func (r *Repository) database() error {
	if err := r.connDb(); err != nil {
		return errors.Wrap(err, "failed to connect database")
	}
	return r.db.Raw("SELECT DATABASE()").Scan(&r.dbInfo.database).Error
}

func (r *Repository) tables() ([]string, error) {
	if err := r.connDb(); err != nil {
		return nil, errors.Wrap(err, "failed to connect database")
	}
	var tables []string
	err := r.db.Raw("SHOW TABLES").Scan(&tables).Error
	return tables, err
}

func (r *Repository) tableColumns(table string) ([]*TableColumns, error) {
	if err := r.connDb(); err != nil {
		return nil, errors.Wrap(err, "failed to connect database")
	}

	var columns []*TableColumns
	err := r.db.Raw("SHOW FULL COLUMNS FROM " + r.Prefix + table).Scan(&columns).Error
	return columns, err
}

func (r *Repository) Generate() error {
	if err := r.database(); err != nil {
		return errors.Wrap(err, "failed to get database name")
	}
	tables, err := r.tables()
	if err != nil {
		return errors.Wrap(err, "failed to get tables")
	}

	for key, value := range r.TableFields {
		generate := keywords.New()
		f := generate.NewGroup()
		f.AddPackage(filepath.Base(r.Path))
		f.NewImport().AddPath("gorm.io/gorm").
			AddPath("log/slog").
			AddPath("time").AddPath("sync").
			AddPath("gitlab.12301.test/gopkg/generic-pkg/iMysql")
		tableUpperCamelCase := ToUpperCamelCase(key)
		tableLowerCamelCase := ToLowerCamelCase(key)
		ok := slices.Contains(tables, key)
		if !ok {
			return errors.Errorf("table %s not found", key)
		}
		// 添加表的结构体
		s := f.NewStruct(tableUpperCamelCase)
		// 获取表字段类型
		columns, err := r.tableColumns(key)
		if err != nil {
			return errors.Wrap(err, "failed to get table columns")
		}
		for _, v := range columns {
			s.AddFieldWithTag(ToUpperCamelCase(v.Field), r.columnType(v.Type), r.tag(v))
		}
		f.NewFunction("TableName").WithReceiver(GetInitials(key), "*"+tableUpperCamelCase).
			AddResult("t", "string").AddBody(keywords.String(`return "` + key + `"`))

		suffix := "Repository"
		repo := tableLowerCamelCase + suffix
		once := tableLowerCamelCase + "Once"
		f.NewStruct(tableUpperCamelCase+suffix).AddField("db", "*gorm.DB")
		f.NewVar().AddDecl(repo, "*"+tableUpperCamelCase+suffix).
			AddDecl(once, "sync.Once")

		f.NewFunction("New"+tableUpperCamelCase+suffix).AddResult("", "*"+tableUpperCamelCase+suffix).
			AddBody(
				keywords.String(tableLowerCamelCase+"Once.Do(func() {"),
				keywords.String(`conn, err := iMysql.Conn("%s")`, r.dbInfo.database),
				keywords.If(keywords.String("err != nil")).AddBody(
					keywords.String(`slog.With(slog.With("database", "%s")).With("err", err).Error("数据库连接失败")`, r.dbInfo.database),
					keywords.String("return"),
				),
				keywords.String(repo+" = &"+tableUpperCamelCase+suffix+"{"),
				keywords.String("db: conn,"),
				keywords.String("}"),
				keywords.String(`})`),
				keywords.String(`return `+repo),
			)

		// 如果字段存在的话，那么创建查询的scope
		for _, v := range value {
			columnType := r.tableFieldType(columns, v)
			f.NewFunction("scope"+ToUpperCamelCase(v)).
				WithReceiver("r", "*"+tableUpperCamelCase+suffix).AddResult("", "func(*gorm.DB) *gorm.DB").
				AddParameter(v, r.columnType(columnType)).
				AddBody(keywords.String(`return func(db *gorm.DB) *gorm.DB {`),
					keywords.If(keywords.String("%s %s", v, r.express(columnType))).AddBody(
						keywords.String(`return db.Where("%s = ?", `+v+`)`, v),
					),
					keywords.String(`return db`),
					keywords.String(`}`),
				)
		}

		if r.IsCreate {
			f.NewFunction("CreateByModel").WithReceiver("r", "*"+tableUpperCamelCase+suffix).AddResult("", "error").
				AddParameter("model", "*"+tableUpperCamelCase).AddBody(keywords.String(`return r.db.Model(&%s{}).Create(model).Error`, tableUpperCamelCase))
		}

		if r.IsUpdate {
			f.NewFunction("UpdateByModel").WithReceiver("r", "*"+tableUpperCamelCase+suffix).AddResult("", "error").
				AddParameter("id", "int").
				AddParameter("model", "*"+tableUpperCamelCase).
				AddBody(keywords.String(`return r.db.Model(&%s{}).Where("id = ?", id).Updates(model).Error`, tableUpperCamelCase))
		}

		// 如果不存在那么就创建目录
		if ok = fileutil.IsExist(r.Path); !ok {
			if err = fileutil.CreateDir(r.Path); err != nil {
				return errors.Wrap(err, "failed to create directory")
			}
		}
		p := fmt.Sprintf("%s/%s.go", r.Path, key)
		if err = generate.WriteFile(p); err != nil {
			return errors.Wrap(err, "failed to write file")
		}
		cmd := exec.Command("go", "fmt", p)
		if err = cmd.Run(); err != nil {
			return errors.Wrap(err, "failed to format code")
		}
	}
	return nil
}

func (r *Repository) tag(column *TableColumns) string {
	str := fmt.Sprintf("gorm:\"column:%s;type:%s", column.Field, column.Type)

	if column.Key == "" && r.columnType(column.Type) == "string" {
		str += ";default:''"
	} else {
		str += fmt.Sprintf(";default:%s", column.Default)
	}
	if column.Key == "UNI" {
		str += ";UNIQUE"
	}
	if column.Key == "PRI" {
		str += ";PRIMARY KEY"
	}
	if column.Extra == "auto_increment" {
		str += ";AUTO_INCREMENT"
	}
	if column.Null == "NO" {
		str += ";NOT NULL"
	}
	if column.Comment != "" {
		str += fmt.Sprintf(";comment:'%s'", column.Comment)
	}
	return str + "\""
}

func (r *Repository) tableFieldType(columns []*TableColumns, f string) string {
	for _, v := range columns {
		if f == v.Field {
			return v.Type
		}
	}
	return ""
}

func (r *Repository) express(t string) string {
	for key, value := range msKeywordsToGoKeywordsMap {
		if strings.Contains(t, key) {
			return value.Express
		}
	}
	return `!=""`
}

func (r *Repository) columnType(t string) string {
	for key, value := range msKeywordsToGoKeywordsMap {
		if strings.Contains(t, key) {
			return value.Keywords
		}
	}
	return "string"
}
