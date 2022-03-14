# stos
为两个结构体的相似字段生成赋值代码

## 安装
``````
go get github.com/buhuiqizi/stos
go install github.com/buhuiqizi/stos
``````

## 作用

程序中可能会存在两个结构体，只有些许字段不同，并且需要互相转换。手动写太麻烦，所以就有了这个小工具。

工具会去掉字段名或者指定Tag值中的下划线，然后将其转换为小写，并作为key映射真实的字段名，然后使用另一个结构体的key去进行匹配，输出赋值代码。

比如：

```
type Student struct {
	StudentName 	`json:"Name"`
}

type StudentParam struct {
	Name 			`json:"m_name"`
}
```

会生成向下面这样的映射：

```
var StudentField   = [][]string{{"studentname", "name"}}
var StudentMap 	   = map[string]string {
	"studentname": "StudentName",
	"name"		 : "StudentName",
}

var StudentParamField = [][]string{{"name", "mname"}}
var StudentParamMap   = map[string]string {
	"name"		 : "Name",
	"mname"		 : "Name",
}
```

生成 Student => StudentParam 的代码时，会先用 studentname 去 StudentParamMap 中匹配，匹配成功则生成代码，然后进行下一个字段的匹配。匹配失败则使用 name 去 StudentParamMap  中匹配。如果使用 studentname 和 name 都没有匹配成功，那么会放入到 notUsed 数组，并统一在最后输出。

## 使用

```
type Student struct {
	StudentsName	 		string	`json:"Name"`
	Age 					int
	Class 					int
}

type StudentParam struct {
	Name					string
	Age 					int
	Class 					int

	Offset 					int
	Limit 					int
}
```

#### 方式1：直接执行命令：

```
stos -left=testData.go/Student -righ=testData.go/StudentParam -tags=json
```

#### 方式2：配置好注释

在结构上面配置好`go:generate`注释

```
//go:generate stos -left=testData.go/Student -righ=testData.go/StudentParam -tags=json
type Student struct {
	StudentsName	 		string	`json:"Name"`
	Age 					int
......
```

然后在文件目录下执行：

```
go generate
```

### 生成的数据

```
func Student_StudentParam(_student *Student, _studentparam *StudentParam) {
	_student.StudentsName	= _studentparam.Name
	_student.Age			= _studentparam.Age
	_student.Class			= _studentparam.Class
}

func StudentParam_Student(_studentparam *StudentParam, _student *Student) {
	_studentparam.Name		= _student.StudentsName
	_studentparam.Age		= _student.Age
	_studentparam.Class		= _student.Class

	// _studentparam not use field:
	// _studentparam.Offset
	// _studentparam.Limit
}
```

