package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

type Config struct {
	LeftTypePath 	string
	RighTypePath 	string

	Tags 			[]string
	RmUnder  		bool
}

type CirType struct {
	Name     		string
	File     		*ast.File
	FilePath 		string

	Fields   		[][]string
	FieldMap 		map[string]string
	FiledMaxLen 	int

	Conf 			*Config
}

func NewCirType(typePath string, conf *Config) *CirType {
	return &CirType{
		Name:     path.Base(typePath),
		FilePath: path.Dir(typePath),
		FieldMap: map[string]string{},
		Conf: 	  conf,
	}
}

type Generator struct {
	buf bytes.Buffer
}

type ParseStorage struct {
	FileMap map[string]*ast.File
}

var (
	leftType   = flag.String("left", "", "path/type, example:./student.go/Student")
	righType   = flag.String("righ", "", "path/type, example:./student.go/Student")
	tags 	   = flag.String("tags", "", "tags     , example:json,orm")
)

// Usage is a replacement usage function for the flags package.
func Usage() {
	fmt.Fprintf(os.Stderr, "Usage of stos:\n")
	fmt.Fprintf(os.Stderr, "\tstos -left=xxx -righ=xxx\n")
	fmt.Fprintf(os.Stderr, "\tstos -left=xxx -righ=xxx -tag=json,orm\n")
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

func getConfig() *Config {
	conf := &Config{}

	if (*leftType) == "" ||  (*righType) == "" {
		flag.Usage()
		os.Exit(1)
	}

	conf.LeftTypePath = *leftType
	conf.RighTypePath = *righType

	if len(*tags) != 0 {
		tagSlice := strings.Split(*tags, ",")
		for i := range tagSlice {
			if tagSlice[i] != "" {
				conf.Tags = append(conf.Tags, tagSlice[i])
			}
		}
	}

	conf.RmUnder = true

	return conf
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("stos: ")

	flag.Usage = Usage
	flag.Parse()

	conf := getConfig()

	leftCt := NewCirType(conf.LeftTypePath, conf)
	righCt := NewCirType(conf.RighTypePath, conf)

	g := &Generator{
		buf: bytes.Buffer{},
	}

	parseStor := &ParseStorage{
		FileMap: map[string]*ast.File{},
	}

	leftCt.File = parseStor.Parse(leftCt.FilePath)
	righCt.File = parseStor.Parse(righCt.FilePath)

	parseStruct(leftCt)
	parseStruct(righCt)

	g.generate(leftCt, righCt, 0)
}

func (prStor *ParseStorage) Parse(FilePath string) *ast.File {
	if prStor.FileMap[FilePath] != nil {
		return prStor.FileMap[FilePath]
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, FilePath, nil, 0)
	if err != nil {
		log.Fatal(fmt.Errorf("ParseFile fail..., filePath - %s,  err - %w", FilePath, err))
	}

	prStor.FileMap[FilePath] = f
	return f
}

func parseStruct(st *CirType) {
	var (
		typeName		string
		simpName 		string
	)

	for _, dec := range st.File.Decls {
		genDec, ok := dec.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, sc := range genDec.Specs {
			tpSc, ok := sc.(*ast.TypeSpec)
			if !ok || tpSc.Name.Name != st.Name {
				continue
			}

			srTp, ok := tpSc.Type.(*ast.StructType)
			if !ok {
				continue
			}

			for _, fd := range srTp.Fields.List {
				var filed []string
				typeName 	= fd.Names[0].Name

				// 将转换后的字段名和tag值，都作为类型名的key
				simpName 	= tranSimple(typeName, st.Conf.RmUnder)
				filed 		= append(filed, simpName)
				st.FieldMap[simpName] = typeName

				if fd.Tag != nil {
					tagstr := strings.Trim(fd.Tag.Value, "`")
					for _, key := range st.Conf.Tags {
						if tag, ok := TagLookup(tagstr, key); ok {

							simpName 	= tranSimple(tag, st.Conf.RmUnder)
							filed 		= append(filed, simpName)
							st.FieldMap[simpName] = typeName
						}
					}
				}

				// 保存最长字段长度，用于对齐
				if len(typeName) > st.FiledMaxLen {
					st.FiledMaxLen = len(typeName)
				}
				st.Fields = append(st.Fields, filed)
			}
		}

	}
}

func tranSimple(name string, rmUnder bool) string {
	name = strings.ToLower(name)
	if !rmUnder {
		return name
	}

	var newname []byte
	for i := range name {
		if name[i] != '_' {
			newname = append(newname, name[i])
		}
	}
	return string(newname)
}

func (g *Generator) generate(leftCt, righCt *CirType, mode int) {
	outFileName := fmt.Sprintf("%s_%s.go", leftCt.Name, righCt.Name)

	g.generateFunc(leftCt, righCt)
	g.Printf("\n")
	g.generateFunc(righCt, leftCt)

	src := g.format()

	err := ioutil.WriteFile(outFileName, src, 0644)
	if err != nil {
		log.Fatalf("writing output: %s", err)
	}
	log.Printf("endFile - %s", outFileName)
}

func (g *Generator) generateFunc(leftCt, righCt *CirType) {
	var (
		leftFieldName string
		righFieldName string
	)

	leftObjName := "_" + strings.ToLower(leftCt.Name)
	righObjName := "_" + strings.ToLower(righCt.Name)

	g.Printf("func %s_%s(%s *%s, %s *%s) {\n", leftCt.Name, righCt.Name,
		leftObjName, leftCt.Name, righObjName, righCt.Name)

	var notUse []string
	for i := range leftCt.Fields {

		leftFieldName = leftCt.FieldMap[leftCt.Fields[i][0]]

		for _, simpName := range leftCt.Fields[i] {
			righFieldName = righCt.FieldMap[simpName]

			if righFieldName != "" {
				g.Printf("\t%s.%s%s= %s.%s\n", leftObjName, leftFieldName,
					needTabs(len(leftObjName)+1+leftCt.FiledMaxLen, len(leftObjName)+1+len(leftFieldName)),
					righObjName, righFieldName)
				goto Next
			}
		}

		notUse = append(notUse, leftFieldName)
		Next:
	}

	if len(notUse) > 0 {
		g.Printf("\n\t// %s not use field:\n", leftObjName)
		for _, fieldName := range notUse {
			g.Printf("\t// %s.%s\n", leftObjName, fieldName)
		}
	}

	g.Printf("}\n")
}

func (g *Generator) format() []byte {
	return g.buf.Bytes()
}

func (g *Generator) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.buf, format, args...)
}

func needTabs(max int, now int) string {
	tabs := max/4+1
	n := (tabs*4-1-now)/4+1
	return getTab(n)
}

var _tabStr = "\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t\t"
func getTab(n int) string {
	return _tabStr[:n]
}


func TagLookup(tag string, key string) (value string, ok bool) {
	for tag != "" {
		// Skip leading space.
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// Scan to colon. A space, a quote or a control character is a syntax error.
		// Strictly speaking, control chars include the range [0x7f, 0x9f], not just
		// [0x00, 0x1f], but in practice, we ignore the multi-byte control characters
		// as it is simpler to inspect the tag's bytes than the tag's runes.
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// Scan quoted string to find value.
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		if key == name {
			value, err := strconv.Unquote(qvalue)
			if err != nil {
				break
			}
			return value, true
		}
	}
	return "", false
}