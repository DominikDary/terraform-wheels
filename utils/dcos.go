package utils

import (
  "encoding/json"
  "github.com/Masterminds/semver/v3"
)

type DCOSPackageVersion struct {
  Version string
}

type GithubRelease struct {
  TagName string `json:"tag_name"`
  Body    string `json:"body"`
}

func GetLatestDCOSVersion(variant string, defaultVersion string) string {
  buf, err := Download("https://versions.d2iq.com/version", WithDefaults).EventuallyReadAll()
  if err != nil {
    return defaultVersion
  }

  data := make(map[string][]DCOSPackageVersion)
  err = json.Unmarshal(buf, &data)
  if err != nil {
    return defaultVersion
  }

  pkgs, ok := data[variant]
  if !ok {
    return defaultVersion
  }

  found := pkgs[0]
  foundVer := semver.MustParse(found.Version)

  for _, pkg := range pkgs {
    ver := semver.MustParse(pkg.Version)
    if ver.Compare(foundVer) > 0 {
      foundVer = ver
      found = pkg
    }
  }

  return found.Version
}

func GetLatestModuleVersion(defaultVersion string) string {
  buf, err := Download("https://api.github.com/repos/dcos-terraform/terraform-aws-dcos/releases", WithDefaults).EventuallyReadAll()
  if err != nil {
    return defaultVersion
  }

  var releases []GithubRelease
  err = json.Unmarshal(buf, &releases)
  if err != nil {
    return defaultVersion
  }

  found := defaultVersion
  foundVer := semver.MustParse("0.0.1")

  for _, rls := range releases {
    ver, err := semver.NewVersion(rls.TagName)
    if err != nil {
      continue
    }

    if ver.Compare(foundVer) > 0 {
      foundVer = ver
      found = rls.TagName
    }
  }

  return found
}
