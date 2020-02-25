package utils

import (
  "bytes"
  "encoding/json"
  "fmt"
)

/**
 * Converts the given input object to a JSON string, even if there are errors
 */
func FormatJSON(anyJson interface{}) string {
  if anyJson == nil {
    return "null"
  }

  bt, err := json.Marshal(anyJson)
  if err != nil {
    return fmt.Sprintf("{ <invalid json: %s> }", err.Error())
  }

  // Pretty-print
  var out bytes.Buffer
  err = json.Indent(&out, bt, "", "  ")
  if err != nil {
    return "{ <indent error> }"
  }
  return out.String()
}
