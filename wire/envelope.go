package wire

import (
	"bytes"
	"fmt"
	"fnd/crypto"
	"io"
	"time"

	"fnd.localhost/dwire"
)

type Envelope struct {
	Magic       uint32
	MessageType MessageType
	Timestamp   time.Time
	Message     Message
	Signature   crypto.Signature
}

func NewEnvelope(magic uint32, message Message, signer crypto.Signer) (*Envelope, error) {
	envelope := &Envelope{
		Magic:       magic,
		MessageType: message.MsgType(),
		Timestamp:   time.Now(),
		Message:     message,
	}
	sig, err := signer.Sign(envelope)
	if err != nil {
		return nil, err
	}
	envelope.Signature = sig
	return envelope, nil
}

func (e *Envelope) Equals(other *Envelope) bool {
	return e.Magic == other.Magic &&
		e.MessageType == other.MessageType &&
		e.Timestamp.Unix() == other.Timestamp.Unix() &&
		e.Message.Equals(other.Message) &&
		e.Signature == other.Signature
}

func (e *Envelope) Encode(w io.Writer) error {
	var buf bytes.Buffer
	if err := e.Message.Encode(&buf); err != nil {
		return err
	}

	return dwire.EncodeFields(
		w,
		e.Magic,
		e.MessageType,
		e.Timestamp,
		buf.Bytes(),
		e.Signature,
	)
}

func (e *Envelope) Decode(r io.Reader) error {
	var msgBuf []byte
	err := dwire.DecodeFields(
		r,
		&e.Magic,
		&e.MessageType,
		&e.Timestamp,
		&msgBuf,
		&e.Signature,
	)
	if err != nil {
		return err
	}

	var msg Message
	switch e.MessageType {
	case MessageTypeHello:
		msg = &Hello{}
	case MessageTypeHelloAck:
		msg = &HelloAck{}
	case MessageTypePing:
		msg = &Ping{}
	case MessageTypeNameUpdate:
		msg = &NameUpdate{}
	case MessageTypeBlobUpdate:
		msg = &BlobUpdate{}
	case MessageTypeBlobNilUpdate:
		msg = &BlobNilUpdate{}
	case MessageTypeNameNilUpdate:
		msg = &NameNilUpdate{}
	case MessageTypeBlobReq:
		msg = &BlobReq{}
	case MessageTypeBlobRes:
		msg = &BlobRes{}
	case MessageTypeNameReq:
		msg = &NameReq{}
	case MessageTypeNameRes:
		msg = &NameRes{}
	case MessageTypePeerReq:
		msg = &PeerReq{}
	case MessageTypePeerRes:
		msg = &PeerRes{}
	case MessageTypeNameUpdateReq:
		msg = &NameUpdateReq{}
	case MessageTypeBlobUpdateReq:
		msg = &BlobUpdateReq{}
	case MessageTypeBlobEquivocationProof:
		msg = &BlobEquivocationProof{}
	case MessageTypeNameEquivocationProof:
		msg = &NameEquivocationProof{}
	default:
		return fmt.Errorf("invalid message type: %d", e.MessageType)
	}

	err = msg.Decode(bytes.NewReader(msgBuf))
	if err != nil {
		return err
	}
	e.Message = msg
	return nil
}

func (e *Envelope) Hash() (crypto.Hash, error) {
	var msgBuf bytes.Buffer
	if err := e.Message.Encode(&msgBuf); err != nil {
		return crypto.ZeroHash, err
	}

	var buf bytes.Buffer
	err := dwire.EncodeFields(
		&buf,
		e.Magic,
		e.MessageType,
		e.Timestamp,
		buf.Bytes(),
	)
	if err != nil {
		return crypto.ZeroHash, err
	}
	return crypto.Blake2B256(buf.Bytes()), nil
}
