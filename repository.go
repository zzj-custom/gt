package template

import (
	"fmt"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gt/g"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

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
		generate := g.New()
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
			s.AddField(ToUpperCamelCase(v.Field), r.columnType(v.Type))
		}
		f.NewFunction("TableName").WithReceiver(GetInitials(key), "*"+tableUpperCamelCase).
			AddResult("t", "string").AddBody(g.String(`return "` + key + `"`))

		suffix := "Repository"
		repo := tableLowerCamelCase + suffix
		once := tableLowerCamelCase + "Once"
		f.NewStruct(tableUpperCamelCase+suffix).AddField("db", "*gorm.DB")
		f.NewVar().AddDecl(repo, "*"+tableUpperCamelCase+suffix).
			AddDecl(once, "sync.Once")

		f.NewFunction("New"+tableUpperCamelCase+suffix).AddResult("", "*"+tableUpperCamelCase+suffix).
			AddBody(
				g.String(tableLowerCamelCase+"Once.Do(func() {"),
				g.String(`conn, err := iMysql.Conn("%s")`, r.dbInfo.database),
				g.If(g.String("err != nil")).AddBody(
					g.String(`slog.With(slog.With("database", "%s")).With("err", err).Error("数据库连接失败")`, r.dbInfo.database),
					g.String("return"),
				),
				g.String(repo+" = &"+tableUpperCamelCase+suffix+"{"),
				g.String("db: conn,"),
				g.String("}"),
				g.String(`})`),
				g.String(`return `+repo),
			)

		// 如果字段存在的话，那么创建查询的scope
		for _, v := range value {
			columnType := r.tableFieldType(columns, v)
			f.NewFunction("scope"+ToUpperCamelCase(v)).
				WithReceiver("r", "*"+tableUpperCamelCase+suffix).AddResult("", "func(*gorm.DB) *gorm.DB").
				AddParameter(v, r.columnType(columnType)).
				AddBody(g.String(`return func(db *gorm.DB) *gorm.DB {`),
					g.If(g.String("%s %s", v, r.express(columnType))).AddBody(
						g.String(`return db.Where("%s = ?", `+v+`)`, v),
					),
					g.String(`return db`),
					g.String(`}`),
				)
		}

		if r.IsCreate {
			f.NewFunction("CreateByModel").WithReceiver("r", "*"+tableUpperCamelCase+suffix).AddResult("", "error").
				AddParameter("model", "*"+tableUpperCamelCase).AddBody(g.String(`return r.db.Model(&%s{}).Create(data).Error`, tableUpperCamelCase))
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

func (r *Repository) tableFieldType(columns []*TableColumns, f string) string {
	for _, v := range columns {
		if f == v.Field {
			return v.Type
		}
	}
	return ""
}

func (r *Repository) express(t string) string {
	if strings.Contains(t, "int") {
		return ">0"
	} else if strings.Contains(t, "varchar") || strings.Contains(t, "text") {
		return `!=""`
	}
	return `!=""`
}

func (r *Repository) columnType(t string) string {
	// TODO 待优化，待补充字段类型
	if t == "" {
		return "string"
	}

	if strings.Contains(t, "int") {
		return "int"
	}

	if strings.Contains(t, "varchar") {
		return "string"
	}

	if strings.Contains(t, "date") {
		return "time.Time"
	}
	return "string"
}
