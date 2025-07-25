//go:build bench
// +build bench

// Code generated by xdrgen. DO NOT EDIT.
// Source: benchmark_autogen_test.go
// Generated 12 XDR types

package xdr_bench

import (
	"fmt"
	"github.com/tempusfrangit/go-xdr"

	"unsafe"
)

func (v *BenchmarkPerson) Encode(enc *xdr.Encoder) error {

	if err := enc.EncodeUint32(v.ID); err != nil {
		return fmt.Errorf("failed to encode ID: %w", err)
	}

	if err := enc.EncodeString(v.Name); err != nil {
		return fmt.Errorf("failed to encode Name: %w", err)
	}

	if err := enc.EncodeUint32(v.Age); err != nil {
		return fmt.Errorf("failed to encode Age: %w", err)
	}

	if err := enc.EncodeString(v.Email); err != nil {
		return fmt.Errorf("failed to encode Email: %w", err)
	}

	return nil
}

func (v *BenchmarkPerson) Decode(dec *xdr.Decoder) error {

	tempID, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode ID: %w", err)
	}
	v.ID = tempID

	tempName, err := dec.DecodeString()
	if err != nil {
		return fmt.Errorf("failed to decode Name: %w", err)
	}
	v.Name = tempName

	tempAge, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode Age: %w", err)
	}
	v.Age = tempAge

	tempEmail, err := dec.DecodeString()
	if err != nil {
		return fmt.Errorf("failed to decode Email: %w", err)
	}
	v.Email = tempEmail

	return nil
}

var _ xdr.Codec = (*BenchmarkPerson)(nil)

func (v *BenchmarkCompany) Encode(enc *xdr.Encoder) error {

	if err := enc.EncodeString(v.Name); err != nil {
		return fmt.Errorf("failed to encode Name: %w", err)
	}

	if err := enc.EncodeUint32(v.Founded); err != nil {
		return fmt.Errorf("failed to encode Founded: %w", err)
	}

	if err := v.CEO.Encode(enc); err != nil {
		return fmt.Errorf("failed to encode CEO: %w", err)
	}

	// #nosec G115
	if err := enc.EncodeUint32(uint32(len(v.Employees))); err != nil {
		return fmt.Errorf("failed to encode Employees length: %w", err)
	}
	for _, elem := range v.Employees {

		if err := elem.Encode(enc); err != nil {
			return fmt.Errorf("failed to encode element: %w", err)
		}

	}

	return nil
}

func (v *BenchmarkCompany) Decode(dec *xdr.Decoder) error {

	tempName, err := dec.DecodeString()
	if err != nil {
		return fmt.Errorf("failed to decode Name: %w", err)
	}
	v.Name = tempName

	tempFounded, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode Founded: %w", err)
	}
	v.Founded = tempFounded

	if err := v.CEO.Decode(dec); err != nil {
		return fmt.Errorf("failed to decode CEO: %w", err)
	}

	EmployeesLen, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode Employees length: %w", err)
	}
	v.Employees = make([]BenchmarkPerson, EmployeesLen)
	for i := range v.Employees {

		if err := v.Employees[i].Decode(dec); err != nil {
			return fmt.Errorf("failed to decode element: %w", err)
		}

	}

	return nil
}

var _ xdr.Codec = (*BenchmarkCompany)(nil)

func (v *BenchmarkConfig) Encode(enc *xdr.Encoder) error {

	if err := enc.EncodeString(v.Host); err != nil {
		return fmt.Errorf("failed to encode Host: %w", err)
	}

	if err := enc.EncodeUint32(v.Port); err != nil {
		return fmt.Errorf("failed to encode Port: %w", err)
	}

	if err := enc.EncodeBool(v.EnableTLS); err != nil {
		return fmt.Errorf("failed to encode EnableTLS: %w", err)
	}

	if err := enc.EncodeUint64(v.Timeout); err != nil {
		return fmt.Errorf("failed to encode Timeout: %w", err)
	}

	// #nosec G115
	if err := enc.EncodeUint32(uint32(len(v.Features))); err != nil {
		return fmt.Errorf("failed to encode Features length: %w", err)
	}
	for _, elem := range v.Features {

		if err := enc.EncodeString(elem); err != nil {
			return fmt.Errorf("failed to encode element: %w", err)
		}

	}

	if err := enc.EncodeBytes(v.Metadata); err != nil {
		return fmt.Errorf("failed to encode Metadata: %w", err)
	}

	return nil
}

func (v *BenchmarkConfig) Decode(dec *xdr.Decoder) error {

	tempHost, err := dec.DecodeString()
	if err != nil {
		return fmt.Errorf("failed to decode Host: %w", err)
	}
	v.Host = tempHost

	tempPort, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode Port: %w", err)
	}
	v.Port = tempPort

	tempEnableTLS, err := dec.DecodeBool()
	if err != nil {
		return fmt.Errorf("failed to decode EnableTLS: %w", err)
	}
	v.EnableTLS = tempEnableTLS

	tempTimeout, err := dec.DecodeUint64()
	if err != nil {
		return fmt.Errorf("failed to decode Timeout: %w", err)
	}
	v.Timeout = tempTimeout

	FeaturesLen, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode Features length: %w", err)
	}
	v.Features = make([]string, FeaturesLen)
	for i := range v.Features {

		val, err := dec.DecodeString()
		if err != nil {
			return fmt.Errorf("failed to decode element: %w", err)
		}
		v.Features[i] = val

	}

	tempMetadata, err := dec.DecodeBytes()
	if err != nil {
		return fmt.Errorf("failed to decode Metadata: %w", err)
	}
	v.Metadata = tempMetadata

	return nil
}

var _ xdr.Codec = (*BenchmarkConfig)(nil)

func (v *BenchmarkResult) Encode(enc *xdr.Encoder) error {

	if err := enc.EncodeUint32(uint32(v.Status)); err != nil {
		return fmt.Errorf("failed to encode Status: %w", err)
	}

	// Switch based on key for union field Data
	switch v.Status {

	case BenchmarkStatusSuccess:
		if err := enc.EncodeBytes(v.Data); err != nil {
			return fmt.Errorf("failed to encode Data: %w", err)
		}

	default:
		// unknown key - encode nothing

	}

	return nil
}

func (v *BenchmarkResult) Decode(dec *xdr.Decoder) error {

	tempStatus, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode Status: %w", err)
	}
	v.Status = BenchmarkStatus(tempStatus)

	// Switch based on key for union field Data
	switch v.Status {

	case BenchmarkStatusSuccess:
		var err error
		v.Data, err = dec.DecodeBytes()
		if err != nil {
			return fmt.Errorf("failed to decode Data: %w", err)
		}

	default:
		// unknown key - decode nothing

	}

	return nil
}

var _ xdr.Codec = (*BenchmarkResult)(nil)

func (v *BenchmarkSuccessResult) Encode(enc *xdr.Encoder) error {

	if err := enc.EncodeString(v.Message); err != nil {
		return fmt.Errorf("failed to encode Message: %w", err)
	}

	return nil
}

func (v *BenchmarkSuccessResult) Decode(dec *xdr.Decoder) error {

	tempMessage, err := dec.DecodeString()
	if err != nil {
		return fmt.Errorf("failed to decode Message: %w", err)
	}
	v.Message = tempMessage

	return nil
}

// ToUnion converts BenchmarkSuccessResult to BenchmarkResult
func (p *BenchmarkSuccessResult) ToUnion() (*BenchmarkResult, error) {
	buf := make([]byte, 1024) // Initial buffer size
	enc := xdr.NewEncoder(buf)
	if err := p.Encode(enc); err != nil {
		return nil, fmt.Errorf("failed to encode BenchmarkSuccessResult: %w", err)
	}
	data := enc.Bytes()

	return &BenchmarkResult{
		Status: BenchmarkStatusSuccess,
		Data:   data,
	}, nil
}

// EncodeToUnion encodes BenchmarkSuccessResult directly to union format
func (p *BenchmarkSuccessResult) EncodeToUnion(enc *xdr.Encoder) error {

	// Encode discriminant
	if err := enc.EncodeUint32(uint32(BenchmarkStatusSuccess)); err != nil {
		return fmt.Errorf("failed to encode discriminant: %w", err)
	}

	// Encode payload
	if err := p.Encode(enc); err != nil {
		return fmt.Errorf("failed to encode payload: %w", err)
	}

	return nil

}

var _ xdr.Codec = (*BenchmarkSuccessResult)(nil)

func (v *BenchmarkMessage) Encode(enc *xdr.Encoder) error {

	if err := enc.EncodeUint32(uint32(v.Type)); err != nil {
		return fmt.Errorf("failed to encode Type: %w", err)
	}

	// Switch based on key for union field Payload
	switch v.Type {

	case BenchmarkMsgText:
		if err := enc.EncodeBytes(v.Payload); err != nil {
			return fmt.Errorf("failed to encode Payload: %w", err)
		}

	default:
		// unknown key - encode nothing

	}

	return nil
}

func (v *BenchmarkMessage) Decode(dec *xdr.Decoder) error {

	tempType, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode Type: %w", err)
	}
	v.Type = BenchmarkMsgType(tempType)

	// Switch based on key for union field Payload
	switch v.Type {

	case BenchmarkMsgText:
		var err error
		v.Payload, err = dec.DecodeBytes()
		if err != nil {
			return fmt.Errorf("failed to decode Payload: %w", err)
		}

	default:
		// unknown key - decode nothing

	}

	return nil
}

var _ xdr.Codec = (*BenchmarkMessage)(nil)

func (v *BenchmarkTextPayload) Encode(enc *xdr.Encoder) error {

	if err := enc.EncodeString(v.Content); err != nil {
		return fmt.Errorf("failed to encode Content: %w", err)
	}

	if err := enc.EncodeString(v.Sender); err != nil {
		return fmt.Errorf("failed to encode Sender: %w", err)
	}

	return nil
}

func (v *BenchmarkTextPayload) Decode(dec *xdr.Decoder) error {

	tempContent, err := dec.DecodeString()
	if err != nil {
		return fmt.Errorf("failed to decode Content: %w", err)
	}
	v.Content = tempContent

	tempSender, err := dec.DecodeString()
	if err != nil {
		return fmt.Errorf("failed to decode Sender: %w", err)
	}
	v.Sender = tempSender

	return nil
}

// ToUnion converts BenchmarkTextPayload to BenchmarkMessage
func (p *BenchmarkTextPayload) ToUnion() (*BenchmarkMessage, error) {
	buf := make([]byte, 1024) // Initial buffer size
	enc := xdr.NewEncoder(buf)
	if err := p.Encode(enc); err != nil {
		return nil, fmt.Errorf("failed to encode BenchmarkTextPayload: %w", err)
	}
	data := enc.Bytes()

	return &BenchmarkMessage{
		Type:    BenchmarkMsgText,
		Payload: data,
	}, nil
}

// EncodeToUnion encodes BenchmarkTextPayload directly to union format
func (p *BenchmarkTextPayload) EncodeToUnion(enc *xdr.Encoder) error {

	// Encode discriminant
	if err := enc.EncodeUint32(uint32(BenchmarkMsgText)); err != nil {
		return fmt.Errorf("failed to encode discriminant: %w", err)
	}

	// Encode payload
	if err := p.Encode(enc); err != nil {
		return fmt.Errorf("failed to encode payload: %w", err)
	}

	return nil

}

var _ xdr.Codec = (*BenchmarkTextPayload)(nil)

func (v *BenchmarkOperation) Encode(enc *xdr.Encoder) error {

	if err := enc.EncodeUint32(uint32(v.OpType)); err != nil {
		return fmt.Errorf("failed to encode OpType: %w", err)
	}

	// Switch based on key for union field Data
	switch v.OpType {

	case BenchmarkOpRead:
		if err := enc.EncodeBytes(v.Data); err != nil {
			return fmt.Errorf("failed to encode Data: %w", err)
		}

	default:
		// unknown key - encode nothing

	}

	return nil
}

func (v *BenchmarkOperation) Decode(dec *xdr.Decoder) error {

	tempOpType, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode OpType: %w", err)
	}
	v.OpType = BenchmarkOpType(tempOpType)

	// Switch based on key for union field Data
	switch v.OpType {

	case BenchmarkOpRead:
		var err error
		v.Data, err = dec.DecodeBytes()
		if err != nil {
			return fmt.Errorf("failed to decode Data: %w", err)
		}

	default:
		// unknown key - decode nothing

	}

	return nil
}

var _ xdr.Codec = (*BenchmarkOperation)(nil)

func (v *BenchmarkReadResult) Encode(enc *xdr.Encoder) error {

	if err := enc.EncodeBool(v.Success); err != nil {
		return fmt.Errorf("failed to encode Success: %w", err)
	}

	if err := enc.EncodeBytes(v.Data); err != nil {
		return fmt.Errorf("failed to encode Data: %w", err)
	}

	if err := enc.EncodeUint32(v.Size); err != nil {
		return fmt.Errorf("failed to encode Size: %w", err)
	}

	return nil
}

func (v *BenchmarkReadResult) Decode(dec *xdr.Decoder) error {

	tempSuccess, err := dec.DecodeBool()
	if err != nil {
		return fmt.Errorf("failed to decode Success: %w", err)
	}
	v.Success = tempSuccess

	tempData, err := dec.DecodeBytes()
	if err != nil {
		return fmt.Errorf("failed to decode Data: %w", err)
	}
	v.Data = tempData

	tempSize, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode Size: %w", err)
	}
	v.Size = tempSize

	return nil
}

// ToUnion converts BenchmarkReadResult to BenchmarkOperation
func (p *BenchmarkReadResult) ToUnion() (*BenchmarkOperation, error) {
	buf := make([]byte, 1024) // Initial buffer size
	enc := xdr.NewEncoder(buf)
	if err := p.Encode(enc); err != nil {
		return nil, fmt.Errorf("failed to encode BenchmarkReadResult: %w", err)
	}
	data := enc.Bytes()

	return &BenchmarkOperation{
		OpType: BenchmarkOpRead,
		Data:   data,
	}, nil
}

// EncodeToUnion encodes BenchmarkReadResult directly to union format
func (p *BenchmarkReadResult) EncodeToUnion(enc *xdr.Encoder) error {

	// Encode discriminant
	if err := enc.EncodeUint32(uint32(BenchmarkOpRead)); err != nil {
		return fmt.Errorf("failed to encode discriminant: %w", err)
	}

	// Encode payload
	if err := p.Encode(enc); err != nil {
		return fmt.Errorf("failed to encode payload: %w", err)
	}

	return nil

}

var _ xdr.Codec = (*BenchmarkReadResult)(nil)

func (v *BenchmarkNode) Encode(enc *xdr.Encoder) error {
	return v.EncodeWithContext(enc, make(map[unsafe.Pointer]bool))
}

func (v *BenchmarkNode) EncodeWithContext(enc *xdr.Encoder, encodingSet map[unsafe.Pointer]bool) error {
	// Check for encoding loop using pointer address (prevents GC from moving objects)
	ptr := unsafe.Pointer(v)
	if encodingSet[ptr] {
		return fmt.Errorf("encoding loop detected for BenchmarkNode")
	}
	encodingSet[ptr] = true
	defer delete(encodingSet, ptr)

	if err := enc.EncodeUint32(v.ID); err != nil {
		return fmt.Errorf("failed to encode ID: %w", err)
	}

	if err := enc.EncodeString(v.Value); err != nil {
		return fmt.Errorf("failed to encode Value: %w", err)
	}

	// #nosec G115
	if err := enc.EncodeUint32(uint32(len(v.Children))); err != nil {
		return fmt.Errorf("failed to encode Children length: %w", err)
	}
	for _, elem := range v.Children {

		if err := elem.EncodeWithContext(enc, encodingSet); err != nil {
			return fmt.Errorf("failed to encode element: %w", err)
		}

	}

	return nil
}

func (v *BenchmarkNode) Decode(dec *xdr.Decoder) error {

	tempID, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode ID: %w", err)
	}
	v.ID = tempID

	tempValue, err := dec.DecodeString()
	if err != nil {
		return fmt.Errorf("failed to decode Value: %w", err)
	}
	v.Value = tempValue

	ChildrenLen, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode Children length: %w", err)
	}
	v.Children = make([]*BenchmarkNode, ChildrenLen)
	for i := range v.Children {

		// Allocate pointer element before decoding
		v.Children[i] = &BenchmarkNode{}

		if err := v.Children[i].Decode(dec); err != nil {
			return fmt.Errorf("failed to decode element: %w", err)
		}

	}

	return nil
}

var _ xdr.Codec = (*BenchmarkNode)(nil)

func (v *BenchmarkFlexibleData) Encode(enc *xdr.Encoder) error {
	return v.EncodeWithContext(enc, make(map[unsafe.Pointer]bool))
}

func (v *BenchmarkFlexibleData) EncodeWithContext(enc *xdr.Encoder, encodingSet map[unsafe.Pointer]bool) error {
	// Check for encoding loop using pointer address (prevents GC from moving objects)
	ptr := unsafe.Pointer(v)
	if encodingSet[ptr] {
		return fmt.Errorf("encoding loop detected for BenchmarkFlexibleData")
	}
	encodingSet[ptr] = true
	defer delete(encodingSet, ptr)

	if err := enc.EncodeUint32(v.ID); err != nil {
		return fmt.Errorf("failed to encode ID: %w", err)
	}

	if v.Next == nil {
		return fmt.Errorf("pointer field Next is nil")
	}

	if err := v.Next.EncodeWithContext(enc, encodingSet); err != nil {
		return fmt.Errorf("failed to encode Next: %w", err)
	}

	return nil
}

func (v *BenchmarkFlexibleData) Decode(dec *xdr.Decoder) error {

	tempID, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode ID: %w", err)
	}
	v.ID = tempID

	// Allocate pointer field before decoding
	v.Next = &BenchmarkFlexibleData{}

	if err := v.Next.Decode(dec); err != nil {
		return fmt.Errorf("failed to decode Next: %w", err)
	}

	return nil
}

var _ xdr.Codec = (*BenchmarkFlexibleData)(nil)

func (v *BenchmarkSimpleData) Encode(enc *xdr.Encoder) error {

	if err := enc.EncodeUint32(v.ID); err != nil {
		return fmt.Errorf("failed to encode ID: %w", err)
	}

	if err := enc.EncodeString(v.Name); err != nil {
		return fmt.Errorf("failed to encode Name: %w", err)
	}

	if err := enc.EncodeInt64(v.Count); err != nil {
		return fmt.Errorf("failed to encode Count: %w", err)
	}

	return nil
}

func (v *BenchmarkSimpleData) Decode(dec *xdr.Decoder) error {

	tempID, err := dec.DecodeUint32()
	if err != nil {
		return fmt.Errorf("failed to decode ID: %w", err)
	}
	v.ID = tempID

	tempName, err := dec.DecodeString()
	if err != nil {
		return fmt.Errorf("failed to decode Name: %w", err)
	}
	v.Name = tempName

	tempCount, err := dec.DecodeInt64()
	if err != nil {
		return fmt.Errorf("failed to decode Count: %w", err)
	}
	v.Count = tempCount

	return nil
}

var _ xdr.Codec = (*BenchmarkSimpleData)(nil)
