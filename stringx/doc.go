/*
Package stringx provides utilities for string naming convention conversions (camelCase, snake_case, PascalCase, etc.).

The package includes functions for converting between different naming conventions commonly used in programming,
such as camelCase, PascalCase, snake_case, and UPPER_SNAKE_CASE. It also provides helper functions for
capitalizing the first letter of strings.

Cache Mechanism:
The package implements a caching mechanism for conversion results to improve performance when the same
conversions are performed repeatedly. Common conversions like Pascal to snake case are cached internally.

Exported Functions:
  - Pascal2Camel(name string) string - Convert PascalCase to camelCase (e.g., "UserName" → "userName")
  - Camel2Pascal(name string) string - Convert camelCase to PascalCase (e.g., "userName" → "UserName")
  - Pascal2Snake(name string) string - Convert PascalCase to snake_case (e.g., "UserName" → "user_name")
  - Pascal2UpperSnake(name string) string - Convert PascalCase to UPPER_SNAKE_CASE (e.g., "UserName" → "USER_NAME")
  - Camel2UpperSnake(name string) string - Convert camelCase to UPPER_SNAKE_CASE (e.g., "userName" → "USER_NAME")
  - Camel2Snake(name string) string - Convert camelCase to snake_case (e.g., "userName" → "user_name")
  - Snake2Pascal(name string) string - Convert snake_case to PascalCase (e.g., "user_name" → "UserName")
  - Snake2Camel(name string) string - Convert snake_case to camelCase (e.g., "user_name" → "userName")
  - Ucfirst(str string) string - Capitalize the first letter of a string (e.g., "hello" → "Hello")
  - Lcfirst(str string) string - Lowercase the first letter of a string (e.g., "Hello" → "hello")

Examples:
  // Convert from PascalCase to other formats
  result := stringx.Pascal2Snake("UserName")        // "user_name"
  result := stringx.Pascal2Camel("UserName")        // "userName"
  result := stringx.Pascal2UpperSnake("UserName")   // "USER_NAME"

  // Convert from camelCase to other formats
  result := stringx.Camel2Snake("userName")         // "user_name"
  result := stringx.Camel2Pascal("userName")        // "UserName"

  // Convert from snake_case to other formats
  result := stringx.Snake2Pascal("user_name")       // "UserName"
  result := stringx.Snake2Camel("user_name")        // "userName"

  // Capitalization helpers
  result := stringx.Ucfirst("hello")                // "Hello"
  result := stringx.Lcfirst("Hello")                // "hello"
*/
package stringx