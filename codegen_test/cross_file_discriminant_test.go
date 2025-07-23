//go:build ignore

//go:generate ../bin/xdrgen $GOFILE

package codegen_test

// Cross-file discriminant union test
// This tests the scenario where the discriminant type is defined in another file

// +xdr:union,key=Status,default=nil  
type OperationResult struct {
    Status ResultStatus // Discriminant defined in cross_file_discriminant_types.go
    Data   []byte       // Payload data
}

// +xdr:payload,union=OperationResult,discriminant=StatusSuccess
type SuccessPayload struct {
    Message string // Success message
}

// +xdr:payload,union=OperationResult,discriminant=StatusError
type ErrorPayload struct {
    ErrorCode uint32 // Error code
}