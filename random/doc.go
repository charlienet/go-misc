/*
Package random provides various random number generators and random string generation utilities.

The package includes three types of random number generators optimized for different use cases:
- SecureGenerator: Cryptographically secure random numbers using crypto/rand
- NormalGenerator: Standard pseudo-random numbers using math/rand (default)
- FastGenerator: Faster random numbers with less overhead (not explicitly mentioned in source, but based on the architecture)

The package provides both low-level random number generation and high-level utilities for generating
random strings with different character sets.

Exported Functions:
  - Int[T scopeConstraint]() T - Generate a random integer of type T
  - Intn[T scopeConstraint](max T) T - Generate a random integer in range [0, max)
  - IntRange[T scopeConstraint](min, max T) T - Generate a random integer in range [min, max)
  - RandBytes(len int) ([]byte, error) - Generate random bytes of specified length
  - StringScope(str string) *charScope - Create a character scope for random string generation

Exported Variables:
  - SecureGenerator - Cryptographically secure random generator
  - NormalGenerator - Standard pseudo-random generator
  - Uppercase - Character scope for uppercase letters
  - Lowercase - Character scope for lowercase letters
  - Digit - Character scope for digits
  - Nomix - Character scope for non-confusing characters
  - Letter - Character scope for all letters
  - Hex - Character scope for hexadecimal characters
  - AllChars - Character scope for all alphanumeric characters

Character Scope Methods:
  - (scope *charScope) Generate(length int, prefix ...string) string - Generate random string with specified length and optional prefix

Generator Types:
- SecureGenerator: Uses crypto/rand for cryptographically secure random numbers. Slower but suitable for security-sensitive applications.
- NormalGenerator: Uses math/rand with thread-safe operations. Good balance of speed and randomness for general purposes.
- FastGenerator: Optimized for speed (based on implementation patterns, though not explicitly exported).

Examples:
  // Using basic random functions with default generator (NormalGenerator)
  n := random.Int[int]()                    // Random int
  n = random.Intn[int](100)                 // Random int in [0, 100)
  n = random.IntRange[int](10, 20)          // Random int in [10, 20)

  // Using different generators
  n = random.SecureGenerator.Intn(100)      // Secure random in [0, 100)
  n = random.NormalGenerator.Intn(100)      // Normal random in [0, 100)

  // Generating random bytes
  bytes, err := random.RandBytes(16)        // 16 random bytes

  // Generating random strings
  password := random.AllChars.Generate(12)              // 12-char random string
  hexCode := random.Hex.Generate(8)                     // 8-char hex string
  id := random.Nomix.Generate(10, "ID-")                // 10-char non-confusing string with "ID-" prefix

  // Using specific character scopes
  upperOnly := random.Uppercase.Generate(10)            // 10 uppercase letters
  digitsOnly := random.Digit.Generate(6)                // 6 digits
  lettersOnly := random.Letter.Generate(8)              // 8 letters (mixed case)

Generator Selection:
Choose the appropriate generator based on your needs:
- Use SecureGenerator when security is paramount (passwords, tokens, keys)
- Use NormalGenerator for general-purpose randomization (shuffling, games, simulations)
- The FastGenerator (implementation detail) provides fastest performance but less randomness guarantees
*/
package random