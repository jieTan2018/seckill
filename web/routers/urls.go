package routers

func productUrls() {
	prodU := r.Group("product")
	{
		prodU.GET("", productList)
		prodU.GET("/:pid", productDetail)
		prodU.POST("", createProduct)
		prodU.PATCH("/:pid", updateProduct)
		prodU.DELETE("/:pid", deleteProduct)
	}
}

func activityUrls() {
	actiU := r.Group("activity")
	{
		actiU.GET("", activityList)
		actiU.GET("/:aid", activityDetail)
		actiU.POST("", createActivity)
		actiU.PATCH("/:aid", updateActivity)
		actiU.DELETE("/:aid", deleteActivity)
	}
}
