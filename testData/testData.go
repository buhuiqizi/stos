package testData

//  -direct=<> -under=rm 这些参数暂时就不加了

//go:generate stos -left=testData.go/Student -righ=testData.go/StudentParam -tags=json
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
