/*
Package fs provides utility functions for file system operations.

The fs package contains helper functions for common file system tasks such as checking file existence.

Exported Functions:
  - Exist: Checks whether a file or directory exists

Example Usage:

Check if a file exists:
 if fs.Exist("somefile.txt") {
     fmt.Println("File exists")
 } else {
     fmt.Println("File does not exist")
 }

Notes:
  - Currently only provides the Exist function for checking file/directory existence
  - Uses os.Stat internally to determine existence
*/
package fs