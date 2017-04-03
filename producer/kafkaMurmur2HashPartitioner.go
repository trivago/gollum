package producer

import (
	kafka "github.com/Shopify/sarama"
	"unsafe"
)

// Murmur2HashPartitioner implements murmur2 hash to be used for kafka messages.
// Murmur2HashPartitioner satisfies sarama.Partitioner so it can be directly assigned to sarama kafka producer config.
// Note: If the key of the message is nil, the message will be partitioned randomly.
type Murmur2HashPartitioner struct {
	random kafka.Partitioner
}

// NewMurmur2HashPartitioner creates a new sarama partitioner based on the
// murmur2 hash algorithm.
func NewMurmur2HashPartitioner(topic string) kafka.Partitioner {
	p := new(Murmur2HashPartitioner)
	p.random = kafka.NewRandomPartitioner(topic)
	return p
}

// Partition chooses a partition based on the murmur2 hash of the key.
// If no key is given a random parition is chosen.
func (p *Murmur2HashPartitioner) Partition(message *kafka.ProducerMessage, numPartitions int32) (int32, error) {
	if message.Key == nil {
		return p.random.Partition(message, numPartitions)
	}
	key, err := message.Key.Encode()
	if err != nil {
		return -1, err
	}
	// murmur2 implementation based on https://github.com/aappleby/smhasher/blob/master/src/MurmurHash2.cpp
	m := uint32(0x5bd1e995)
	r := uint32(24)
	seed := uint32(0x9747b28c)
	dataLen := uint32(len(key))
	/* Initialize the hash to a 'random' value */

	h := seed ^ dataLen

	/* Mix 4 bytes at a time into the hash */
	data := key
	i := 0
	for {
		if dataLen < 4 {
			break
		}
		k := *(*uint32)(unsafe.Pointer(&data[i*4]))
		k *= m
		k ^= k >> r
		k *= m

		h *= m
		h ^= k
		i++
		dataLen -= 4
	}

	/* Handle the last few bytes of the input array */

	switch dataLen {
	case 3:
		h ^= uint32(data[i*4+2]) << 16
		h ^= uint32(data[i*4+1]) << 8
		h ^= uint32(data[i*4])
		h *= m
	case 2:
		h ^= uint32(data[i*4+1]) << 8
		h ^= uint32(data[i*4])
		h *= m
	case 1:
		h ^= uint32(data[i*4])
		h *= m
	default:
	}

	/* Do a few final mixes of the hash to ensure the last few
	 * bytes are well-incorporated. */

	h ^= h >> 13
	h *= m
	h ^= h >> 15

	// convert h to positive
	partition := (int32(h) & 0x7fffffff) % numPartitions

	return partition, nil

}

// RequiresConsistency always returns true
func (p *Murmur2HashPartitioner) RequiresConsistency() bool {
	return true
}
