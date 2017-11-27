package sg1

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	defSeqNumber = uint32(0)
	defSeqTotal  = uint32(1)
	defData      = []byte{0xde, 0xad, 0xbe, 0xef}
	defDataSize  = uint32(len(defData))
	defPacket    = NewPacket(defSeqNumber, defSeqTotal, defDataSize, defData)
	defCopy      = (*Packet)(nil)
)

func TestPacketCreation(t *testing.T) {
	assert.Equal(t, defSeqNumber, defPacket.SeqNumber)
	assert.Equal(t, defSeqTotal, defPacket.SeqTotal)
	assert.Equal(t, defDataSize, defPacket.DataSize)
	assert.Equal(t, defData, defPacket.Data)
}

func TestPacketCopy(t *testing.T) {
	assert.Nil(t, defCopy)
	defCopy = defPacket.Copy()
	assert.NotNil(t, defCopy)

	assert.Equal(t, defPacket.SeqNumber, defCopy.SeqNumber)
	assert.Equal(t, defPacket.SeqTotal, defCopy.SeqTotal)
	assert.Equal(t, defPacket.DataSize, defCopy.DataSize)
	assert.Equal(t, defPacket.Data, defCopy.Data)
}

func TestPacketRaw(t *testing.T) {
	raw := defPacket.Raw()
	assert.NotNil(t, raw)
	sz := uint32(len(raw))

	assert.Equal(t, uint32(defPacket.HeaderSize())+defPacket.DataSize, sz)

	assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x00}, raw[0:4])  // seq n
	assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x01}, raw[4:8])  // seq total
	assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x04}, raw[8:12]) // data size
	assert.Equal(t, defData, raw[12:])                         // data
}

func TestDecodePacket(t *testing.T) {
	raw := defPacket.Raw()
	assert.NotNil(t, raw)

	p, err := DecodePacket(raw)
	assert.Nil(t, err)
	assert.NotNil(t, p)

	assert.Equal(t, defPacket.SeqNumber, p.SeqNumber)
	assert.Equal(t, defPacket.SeqTotal, p.SeqTotal)
	assert.Equal(t, defPacket.DataSize, p.DataSize)
	assert.Equal(t, defPacket.Data, p.Data)
}

func TestDecodeShortPacket(t *testing.T) {
	raw := []byte{0x00}
	_, err := DecodePacket(raw)
	assert.NotNil(t, err)
}

func TestDecodeMalformedPacket(t *testing.T) {
	p := defPacket.Copy()
	p.DataSize++
	raw := p.Raw()
	_, err := DecodePacket(raw)
	assert.NotNil(t, err)
}
