package utils

import (
  "bytes"
  "encoding/json"
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
    return "{ <invalid json> }"
  }

  // Pretty-print
  var out bytes.Buffer
  err = json.Indent(&out, bt, "", "  ")
  if err != nil {
    return "{ <indent error> }"
  }
  return out.String()
}
