package models

import (
	"gorm.io/gorm"
)

type Product struct {
	ProductId   int    `json:"product_id" gorm:"column:id" form:"pId" binding:"required"`
	ProductName string `json:"product_name" gorm:"column:name" form:"pName" binding:"required"`
	Total       int    `json:"total" form:"total"`
	Status      int    `json:"status" form:"status"`
}

func NewProduct() *Product {
	return &Product{}
}

func (p *Product) TableName() string {
	return "products"
}

func (p *Product) GetProductList() (list *[]Product, err error) {
	list = &[]Product{}
	err = orm.Find(list).Error
	if err != nil {
		logs.Errorf("get product list failed! err:%v", err)
		return
	}
	return
}

func (p *Product) ProductWithId(id string) (prod *Product, err error) {
	prod = &Product{}
	err = orm.First(prod, id).Error
	if err != nil {
		logs.Errorf("get product by id failed! err:%v", err)
		return
	}
	return
}

func (p *Product) AddProduct(prod *Product) (err error) {
	err = orm.Create(prod).Error
	if err != nil {
		logs.Errorf("create product failed! err:%v", err)
		return
	}
	return
}

func (p *Product) UpdateProduct(id string, params map[string]interface{}) (err error) {
	ret := orm.Model(Product{}).Where("id=?", id).Updates(params)
	if ret.RowsAffected == 0 {
		logs.Error("no rows are affected!")
		return gorm.ErrRecordNotFound
	}
	return ret.Error
}

func (p *Product) DeleteProduct(id string) (err error) {
	ret := orm.Unscoped().Where("id=?", id).Delete(Product{})
	if ret.RowsAffected == 0 {
		logs.Error("no rows are affected!")
		return gorm.ErrRecordNotFound
	}
	return ret.Error
}

type productUpdateValid struct {
	ProductId   int    `json:"product_id" gorm:"column:id" form:"pId" binding:"required"`
	ProductName string `json:"product_name" gorm:"column:name" form:"pName"`
	Total       int    `json:"total" form:"total"`
	Status      int    `json:"status" form:"status"`
}

func NewProductUpdateValid() *productUpdateValid {
	return &productUpdateValid{}
}
