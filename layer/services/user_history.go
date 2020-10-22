package services

import (
	"sync"
)

type UserBuyHistory struct {
	sync.RWMutex
	history map[int]int
}

func (p *UserBuyHistory) GetProductBuyCount(productId int) int { // 获取活用购买商品的数量
	p.RLock()
	defer p.RUnlock()

	count, _ := p.history[productId]
	return count
}

func (p *UserBuyHistory) Add(productId, count int) {
	p.Lock()
	defer p.Unlock()

	cur, ok := p.history[productId]
	if !ok {
		cur = count
	} else {
		cur += count
	}

	p.history[productId] = cur
}
