package main

//
//import "github.com/albrow/zoom"
//
//type Category struct {
//	zoom.RandomID
//	Name          string `zoom:"index"`
//	DispatchTAsID []string
//	Description   string
//}
//
//var (
//	Categories *zoom.Collection
//)
//
//func newCategory() *Category {
//	return &Category{
//		Name:          "",
//		DispatchTAsID: []string{},
//		Description:   "",
//	}
//}
//
//func CreateCategories() {
//	_Categories, err := pool.NewCollectionWithOptions(&Category{},
//		zoom.DefaultCollectionOptions.WithIndex(true))
//	if err != nil {
//		// handle error
//		panic(err)
//	}
//
//	Categories = _Categories
//}
//
//func GetCategory(id string) *Category {
//	cat := newCategory()
//	if err := Students.Find(id, cat); err != nil {
//		return nil
//	}
//	return cat
//}
//
////
////func GetAllCategories() *[]Category {
////
////}
