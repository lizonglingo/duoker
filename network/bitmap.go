package network

// bitMap 用来表示子网中 IP 是否被使用.
// 使用位操作简化 IP 管理.
type bitMap struct {
	Bitmap []byte
}

// InitBitMap 初始化特定长度的 bitmap.
func InitBitMap(maxLen int64) *bitMap {
	b := &bitMap{}
	b.Bitmap = make([]byte, maxLen)
	return b
}

// BitExist 判断某个网络地址是否被使用.
func (b *bitMap) BitExist(pos int) bool {
	aIndex := arrIndex(pos) // 获得该地址对应的字节位置
	bIndex := bytePos(pos)  // 获得该地址对应字节中的位的位置
	// b.Bitmap[aIndex]>>bIndex 将对应位移到最低位
	return 1 == 1&(b.Bitmap[aIndex]>>bIndex)
}

// BitSet 将 IP 设置成已经使用.
func (b *bitMap) BitSet(pos int) {
	aIndex := arrIndex(pos)
	bIndex := bytePos(pos)
	b.Bitmap[aIndex] = b.Bitmap[aIndex] | (1 << bIndex)
}

// BitClean 释放不再使用的 IP.
func (b *bitMap) BitClean(pos int) {
	aIndex := arrIndex(pos)
	bIndex := bytePos(pos)
	b.Bitmap[aIndex] = b.Bitmap[aIndex] & (^(1 << bIndex))
}

// 一个字节是 8位
// 一个 IPv4 地址是 32位 对应四个字节
// 第 pos 个地址 / 8 得到它在哪个字节
// 第 pos 个地址 % 8 得到具体是字节的哪一位

// arrIndex 地址在哪一个字节.
func arrIndex(pos int) int {
	return pos / 8
}

// bytePos 地址在这个字节的哪一位.
func bytePos(pos int) int {
	return pos % 8
}
