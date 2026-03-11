package consistenthash

import (
	"hash/crc32"
	"sort"
	"strconv"
)

type Hash func(data []byte) uint32

type Map struct {
	hash     Hash //允许自定义哈希函数
	replicas int
	keys     []int          //哈希环，使用[]int的原因是查找频繁而不是插入频繁，使用切片更节省内存且查询更快。能够更简单实现排序和二分查找
	hashMap  map[int]string //虚拟节点和真实节点的映射
}

func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE //哈希空间为2^32
	}
	return m
}

func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key))) //虚拟节点分布不是严格均匀，但是checksumIEEE的空间是足够大的，冲突概率很低。理论依据：大数定律+好哈希函数
			//假如节点是2，那么虚拟节点就是0+2,1+2,2+2...m.replicas-1+2，也就是2，12，22，32...
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys) //排序原因：一致性哈希核心操作是在哈希环上顺时针找到第一个>=keyHash的节点
}
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}
	hash := int(m.hash([]byte(key))) //计算key的哈希值
	//二分查找法找到第一个>=hash的下标
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	//取模实现环形
	return m.hashMap[m.keys[idx%len(m.keys)]] //这里取余的原因是一个特殊情况：如果二分查找hash值比所有节点的哈希值都大，应该返回第一个节点。
}
