// Package codegen_test provides test cases for cross-file discriminated union support.
package codegen_test

// ResultStatus represents operation status codes for cross-file discriminated unions.
type ResultStatus uint32

const (
    // StatusSuccess indicates successful operation completion.
    StatusSuccess ResultStatus = 0
    // StatusError indicates operation failed with error.
    StatusError ResultStatus = 1
)