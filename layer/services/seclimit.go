package services

type SecLimit struct {
	count   int
	curTime int64
}

func (s *SecLimit) Counter(nowTime int64) int { // 计算访问次数
	if s.curTime != nowTime {
		s.count = 1
		s.curTime = nowTime
		return s.count
	}
	s.count++
	return s.count
}

func (s *SecLimit) Check(nowTime int64) int { // 获取访问次数
	if s.curTime != nowTime {
		return 0
	}

	return s.count
}
