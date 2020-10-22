package services

func serviceRun() error {
	// 初始化处理线程
	err := runProcess()
	if err != nil {
		panic("service run err!")
	}
	return nil
}
