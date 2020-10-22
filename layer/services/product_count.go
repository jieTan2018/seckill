package services

import (
	"sync"
)

type ProductCountMgr struct {
	sync.RWMutex // 抢到时,才会有写操作; 又因为抢到次数远小于查看次数
	productCount map[int]int
}

func NewProductCountMgr() *ProductCountMgr {
	return &ProductCountMgr{
		productCount: make(map[int]int, 128),
	}
}

func (p *ProductCountMgr) Count(productId int) int {
	p.RLock()
	defer p.RUnlock()

	count, _ := p.productCount[productId]
	return count
}

func (p *ProductCountMgr) Add(productId, count int) {
	p.Lock()
	defer p.Unlock()

	cur, ok := p.productCount[productId]
	if !ok {
		cur = count
	} else {
		cur += count
	}

	p.productCount[productId] = cur
}
