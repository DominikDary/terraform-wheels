package plugins

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
)

func ToJson(iface interface{}) string {
  v, _ := json.Marshal(iface)
  return string(v)
}

func ToJsonString(iface interface{}) string {
  v, _ := json.Marshal(iface)
  sv, _ := json.Marshal(string(v))
  return string(sv)
}

func interfaceToLines(iface interface{}, path string, lines []string) []string {
  var ret []string = append(lines, "")
  var segLines []string

  switch v := iface.(type) {
  case []string:
    ret = append(ret, []string{
      `section {`,
      fmt.Sprintf(`  path = "%s"`, path),
      fmt.Sprintf(`  list = [`),
    }...)
    for _, v := range v {
      ret = append(ret, fmt.Sprintf("    %s,", ToJson(v)))
    }
    ret = append(ret, []string{
      fmt.Sprintf(`  ]`),
      `}`,
    }...)

  case map[string]interface{}:

    for k, v := range v {
      switch sv := v.(type) {
      case string:
        segLines = append(segLines, fmt.Sprintf("    %s = %s,", k, ToJson(sv)))
      case int:
        segLines = append(segLines, fmt.Sprintf("    %s = %s,", k, ToJson(sv)))
      case float64:
        segLines = append(segLines, fmt.Sprintf("    %s = %s,", k, ToJson(sv)))
      case bool:
        segLines = append(segLines, fmt.Sprintf("    %s = %s,", k, ToJson(sv)))
      default:
        ret = interfaceToLines(v, path+"."+k, ret)
      }
    }

    if len(segLines) > 0 {
      ret = append(ret, []string{
        `section {`,
        fmt.Sprintf(`  path = "%s"`, path),
        `  map = {`,
      }...)
      ret = append(ret, segLines...)
      ret = append(ret, []string{
        `  }`,
        `}`,
      }...)
    }

  default:
    fmt.Printf("Type: <%T> %#v\n", v, v)
    ret = append(ret, []string{
      `section {`,
      fmt.Sprintf(`  path = "%s"`, path),
      fmt.Sprintf(`  json = <<EOF`),
      fmt.Sprintf(`  %s`, ToJson(iface)),
      fmt.Sprintf(`  EOF`),
      `}`,
    }...)
  }

  return ret
}

func LoadServiceJsonToConfigLines(filename string) ([]string, error) {
  content, err := ioutil.ReadFile(filename)
  if err != nil {
    return nil, err
  }

  var config map[string]interface{}
  err = json.Unmarshal(content, &config)
  if err != nil {
    return nil, err
  }

  var lines []string
  for k, v := range config {
    lines = interfaceToLines(v, k, lines)
  }

  return lines, nil
}
