/*
Package currency provides utilities for monetary calculations and conversions.

The currency package handles conversion between different monetary units (such as yuan and fen) with precision arithmetic using the decimal library to avoid floating-point errors.

Exported Functions:
  - CentToDollar: Converts cents to dollar string representation
  - DollarToCent: Converts dollar string to cents integer
  - FenToYuan: Converts fen to yuan string representation (generic version)
  - YuanToFen: Converts yuan string to fen integer

Example Usage:

Converting cents to dollars:
 result := currency.CentToDollar(123)  // Returns "1.23"

Converting dollars to cents:
 cents := currency.DollarToCent("1.23")  // Returns 123

Using generic function for fen to yuan:
 result := currency.FenToYuan(123)  // Returns "1.23"

Converting yuan to fen:
 fen := currency.YuanToFen("1.23")  // Returns 123

Precision Notes:
  - All calculations use the shopspring/decimal library for precise arithmetic
  - This avoids floating-point precision issues common in financial calculations
  - Results are rounded appropriately according to banking rules (round half to even)
*/
package currency