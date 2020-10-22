package configs

func etcdTestDatas() *[]SecInfoConf {
	var secProductInfos []SecInfoConf
	secProductInfos = append(secProductInfos, SecInfoConf{
		ProductId: 1027,
		StartTime: 1602468000,
		EndTime:   1602469800,
		Status:    0,
		Total:     1000,
		Left:      1000,
	})
	secProductInfos = append(secProductInfos, SecInfoConf{
		ProductId: 1028,
		StartTime: 1602468000,
		EndTime:   1602469800,
		Status:    0,
		Total:     1000,
		Left:      1000,
	})

	return &secProductInfos
}

func etcdTestDatas2() *[]SecInfoConf {
	var secProductInfos []SecInfoConf
	secProductInfos = append(secProductInfos, SecInfoConf{
		ProductId: 1029,
		StartTime: 1602468000,
		EndTime:   1602469800,
		Status:    0,
		Total:     9999,
		Left:      9999,
	})
	secProductInfos = append(secProductInfos, SecInfoConf{
		ProductId: 1030,
		StartTime: 1602468000,
		EndTime:   1602469800,
		Status:    0,
		Total:     6666,
		Left:      6666,
	})

	return &secProductInfos
}
