package main

//go:generate ../../bin/xdrgen types.go

import (
	"fmt"
	"log"

	"github.com/tempusfrangit/go-xdr"
)

func main() {
	fmt.Println("=== Auto-Generated XDR Encoding/Decoding Example ===")

	// Create a person instance
	person := &Person{
		ID:    1001,
		Name:  "Alice Johnson",
		Age:   30,
		Email: "alice@example.com",
	}

	fmt.Printf("\n1. Original person: %+v\n", person)

	// Marshal using auto-generated methods
	data, err := xdr.Marshal(person)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("2. Marshaled %d bytes: %x\n", len(data), data)

	// Unmarshal using auto-generated methods
	var decoded Person
	err = xdr.Unmarshal(data, &decoded)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("3. Unmarshaled person: %+v\n", decoded)

	// Verify round-trip
	if person.ID == decoded.ID && person.Name == decoded.Name &&
		person.Age == decoded.Age && person.Email == decoded.Email {
		fmt.Println("Round-trip successful!")
	} else {
		fmt.Println("Round-trip failed!")
	}

	// Example with nested structures
	fmt.Println("\n4. Nested structure example...")
	company := &Company{
		Name:    "Tech Corp",
		Founded: 2010,
		CEO: Person{
			ID:    1,
			Name:  "John Smith",
			Age:   45,
			Email: "john@techcorp.com",
		},
		Employees: []Person{
			{ID: 2, Name: "Jane Doe", Age: 28, Email: "jane@techcorp.com"},
			{ID: 3, Name: "Bob Wilson", Age: 35, Email: "bob@techcorp.com"},
		},
	}

	fmt.Printf("Original company: %+v\n", company)

	// Marshal nested structure
	companyData, err := xdr.Marshal(company)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Marshaled company (%d bytes)\n", len(companyData))

	// Unmarshal nested structure
	var decodedCompany Company
	err = xdr.Unmarshal(companyData, &decodedCompany)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Unmarshaled company: %+v\n", decodedCompany)
	fmt.Printf("CEO: %+v\n", decodedCompany.CEO)
	fmt.Printf("Employees (%d):\n", len(decodedCompany.Employees))
	for i, emp := range decodedCompany.Employees {
		fmt.Printf("  %d: %+v\n", i+1, emp)
	}

	// Example with configuration structure
	fmt.Println("\n5. Configuration structure example...")
	config := &ServerConfig{
		Host:       "localhost",
		Port:       8080,
		EnableTLS:  true,
		MaxClients: 100,
		Timeout:    30000,
		LogLevel:   "info",
		Features:   []string{"auth", "logging", "metrics"},
		Metadata:   []byte("server-v1.0"),
	}

	fmt.Printf("Original config: %+v\n", config)

	configData, err := xdr.Marshal(config)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Marshaled config (%d bytes)\n", len(configData))

	var decodedConfig ServerConfig
	err = xdr.Unmarshal(configData, &decodedConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Unmarshaled config: %+v\n", decodedConfig)

	fmt.Println("\n=== Auto-generation example complete ===")
}
