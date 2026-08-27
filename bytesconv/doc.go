/*
Package bytesconv provides zero-copy string and byte slice conversion utilities along with a BytesResult type.

The package offers functions for converting between strings and byte slices without memory allocation,
which can significantly improve performance in certain scenarios. However, these functions use unsafe
operations and come with important safety considerations that must be understood before use.

Safety Warning:
The zero-copy conversion functions (StringToBytes and BytesToString) use unsafe operations to avoid
memory allocation. This means the returned slice or string shares memory with the original data.
Modifying the result of StringToBytes will cause undefined behavior since strings in Go are immutable.
Similarly, if the original byte slice passed to BytesToString is modified after the conversion, the
resulting string may also change, leading to undefined behavior. Only use these functions when you
can guarantee the data won't be modified or when you only need read-only access.

Exported Functions:
  - StringToBytes(s string) []byte - Convert string to byte slice without memory allocation
  - BytesToString(b []byte) string - Convert byte slice to string without memory allocation

BytesResult Type:
The package also defines a BytesResult type with the following methods:
  - FromString(s string) BytesResult - Create BytesResult from string
  - FromBytes(b []byte) BytesResult - Create BytesResult from byte slice
  - FromHexString(s string) (BytesResult, error) - Create BytesResult from hexadecimal string
  - FromBase64String(s string) (BytesResult, error) - Create BytesResult from Base64 string
  - (r BytesResult) Hex() string - Convert to hexadecimal string representation
  - (r BytesResult) UppercaseHex() string - Convert to uppercase hexadecimal string representation
  - (r BytesResult) Base64() string - Convert to Base64 string representation
  - (r BytesResult) Bytes() []byte - Get underlying byte slice
  - (r BytesResult) String() string - Get string representation (hexadecimal)
  - (r BytesResult) Open() io.Reader - Create an io.Reader from the data

Examples:
  // Zero-copy string to bytes conversion (use with caution!)
  s := "hello world"
  b := bytesconv.StringToBytes(s)  // Shares memory with s, do not modify b!
  fmt.Printf("%s\n", string(b))    // "hello world"

  // Zero-copy bytes to string conversion (use with caution!)
  b := []byte("hello world")
  s := bytesconv.BytesToString(b)  // Shares memory with b, ensure b is not modified!
  fmt.Printf("%s\n", s)            // "hello world"

  // Using BytesResult type
  result := bytesconv.FromString("hello")
  hexStr := result.Hex()           // Convert to hex: "68656c6c6f"
  base64Str := result.Base64()     // Convert to base64: "aGVsbG8="
  
  // Creating from different formats
  fromHex, err := bytesconv.FromHexString("68656c6c6f")
  fromBase64, err := bytesconv.FromBase64String("aGVsbG8=")

Security Considerations:
Because the conversion functions use unsafe operations, they bypass Go's type safety. This can lead
to undefined behavior if the resulting data is modified inappropriately. Only use these functions
when performance is critical and you fully understand the implications of sharing memory between
strings and byte slices.
*/
package bytesconv