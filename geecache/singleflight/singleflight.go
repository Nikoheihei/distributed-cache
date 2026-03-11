package singleflight

import "sync"

type call struct {
	wg  sync.WaitGroup //只允许一个人查其他人等待查询结果
	val interface{}    //数据放在这
	err error
}

type Group struct {
	mu sync.Mutex       //并发保护
	m  map[string]*call //string是请求的键
}

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		g.mu.Unlock()
		c.wg.Wait()
		return c.val, c.err
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.mu.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	//查询完毕可以删除，因为下一时刻对同一key的请求实际上和现在的并不并发
	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}
