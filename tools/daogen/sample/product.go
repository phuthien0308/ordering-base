package sample

//go:generate daogen -struct Product -file product.go
type Product struct {
	ID    int64  `sql-col:"id" sql-identifier:"true"`
	Name  string `sql-col:"name" sql-insert:"true" sql-update:"true"`
	Price int64  `sql-col:"price" sql-insert:"true" sql-update:"true"`
}
