package wire

import (
	"fnd/crypto"
	"io"

	"fnd.localhost/dwire"
)

type Message interface {
	crypto.Hasher
	dwire.EncodeDecoder
	MsgType() MessageType
	Equals(other Message) bool
}

type MessageType uint16

const (
	MessageTypeHello MessageType = iota
	MessageTypeHelloAck
	MessageTypePing
	MessageTypeNameUpdate
	MessageTypeBlobUpdate
	MessageTypeBlobNilUpdate
	MessageTypeNameNilUpdate
	MessageTypeBlobReq
	MessageTypeBlobRes
	MessageTypeNameReq
	MessageTypeNameRes
	MessageTypePeerReq
	MessageTypePeerRes
	MessageTypeBlobUpdateReq
	MessageTypeNameUpdateReq
	MessageTypeEquivocationProof
)

func (t MessageType) String() string {
	switch t {
	case MessageTypeHello:
		return "Hello"
	case MessageTypeHelloAck:
		return "HelloAck"
	case MessageTypePing:
		return "Ping"
	case MessageTypeNameUpdate:
		return "NameUpdate"
	case MessageTypeBlobUpdate:
		return "BlobUpdate"
	case MessageTypeBlobNilUpdate:
		return "BlobNilUpdate"
	case MessageTypeNameNilUpdate:
		return "NameNilUpdate"
	case MessageTypeBlobReq:
		return "BlobReq"
	case MessageTypeBlobRes:
		return "BlobRes"
	case MessageTypeNameReq:
		return "NameReq"
	case MessageTypeNameRes:
		return "NameRes"
	case MessageTypePeerReq:
		return "PeerReq"
	case MessageTypePeerRes:
		return "PeerRes"
	case MessageTypeNameUpdateReq:
		return "NameUpdateReq"
	case MessageTypeBlobUpdateReq:
		return "BlobUpdateReq"
	case MessageTypeEquivocationProof:
		return "EquivocationProof"
	default:
		return "unknown"
	}
}

func (t MessageType) Encode(w io.Writer) error {
	return dwire.EncodeField(w, uint16(t))
}

func (t *MessageType) Decode(r io.Reader) error {
	var decoded uint16
	if err := dwire.DecodeField(r, &decoded); err != nil {
		return err
	}
	*t = MessageType(decoded)
	return nil
}
