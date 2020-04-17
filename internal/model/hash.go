package model


func BKDRHash64(str []byte) uint64 {
    var seed uint64 = 131; // 31 131 1313 13131 131313 etc..
    var hash uint64 = 0;
    for i := 0; i < len(str); i++ {
        hash = hash * seed + uint64(str[i])
    }
    return (hash & 0x7FFFFFFFFFFFFFFF)
}
