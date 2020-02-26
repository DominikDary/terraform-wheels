package utils

import (
  "encoding/json"
  "fmt"
  "os"
  "os/exec"
  "path/filepath"
  "runtime"
  "strings"
  "time"

  "github.com/Masterminds/semver/v3"
)

type LatestVersion struct {
  Version *semver.Version
  URL     string
}

/**
 * Get the latest released version
 */
func GetLatestVersion() (LatestVersion, error) {
  res := LatestVersion{}

  // Download latest version
  byt, err := Download("http://api.github.com/repos/mesosphere-incubator/terraform-launch/releases/latest", WithDefaults).
    EventuallyReadAll()

  // Parse contents
  var dat map[string]interface{}
  if err := json.Unmarshal(byt, &dat); err != nil {
    return res, fmt.Errorf("error parsing version info: %s", err.Error())
  }

  // Extract version
  var ok bool
  var tagName string
  if tagName, ok = dat["tag_name"].(string); !ok {
    return res, fmt.Errorf("invalid version info: missing `tag_name`")
  }
  if !strings.HasPrefix(tagName, "v") {
    return res, fmt.Errorf("invalid tag name")
  }

  // Scan assets
  var downloadUrl string = ""
  var assets []interface{}
  if assets, ok = dat["assets"].([]interface{}); !ok {
    return res, fmt.Errorf("invalid version info: missing `assets`")
  }
  for _, asset := range assets {
    var mapInst map[string]interface{}
    if mapInst, ok = asset.(map[string]interface{}); !ok {
      return res, fmt.Errorf("invalid version info: invalid field `assets`")
    }
    if url, ok := mapInst["browser_download_url"].(string); ok {
      if strings.Contains(url, runtime.GOOS) {
        downloadUrl = url
        break
      }
    }
  }
  if downloadUrl == "" {
    return res, fmt.Errorf("could not find a download URL")
  }

  ver, err := semver.NewVersion(tagName[1:])
  if err != nil {
    return res, fmt.Errorf("Could not parse released version '%s': %s", tagName[1:], err.Error())
  }

  res.Version = ver
  res.URL = downloadUrl

  return res, nil
}

/**
 * Perform upgrade
 */
func PerformUpgrade(newVersion LatestVersion) error {

  // Find the path of our current executable
  replaceTarget, err := os.Executable()
  if err != nil {
    return fmt.Errorf("could not find the location of the tool: %s", err.Error())
  }

  // Rename the current executable
  bakTarget := replaceTarget + ".bak"
  err = os.Rename(replaceTarget, bakTarget)
  if err != nil {
    return fmt.Errorf("could not rename old version: %s", err.Error())
  }

  // Download the new version
  stream := Download(newVersion.URL, WithDefaults).AndShowProgress("")
  if runtime.GOOS == "windows" {
    err = stream.
      EventuallyUnzipOnlyTo(
        filepath.Dir(replaceTarget)+"/",
        []string{filepath.Base(replaceTarget)},
        0)
  } else {
    err = stream.
      AndDecompressIfCompressed().
      EventuallyUntarOnlyTo(
        filepath.Dir(replaceTarget)+"/",
        []string{filepath.Base(replaceTarget)},
        0)
  }
  if err != nil {
    os.Remove(replaceTarget)
    os.Rename(bakTarget, replaceTarget)
    return fmt.Errorf("could not process file stream: %s", err.Error())
  }

  // Make it executable and close it
  os.Chmod(replaceTarget, 0755)

  // Run the new version that is going to remove the backup file
  cmd := exec.Command(replaceTarget, "tool-complete-upgrade", bakTarget)
  err = cmd.Start()
  if err != nil {
    os.Rename(replaceTarget, replaceTarget+".check")
    os.Rename(bakTarget, replaceTarget)
    return fmt.Errorf("could run the new version: %s", err.Error())
  }

  // Completed
  return nil
}

/**
 * Helper function to complete an upgrade process
 */
func CompleteUpgrade(bakTarget string) {
  // Wait for the other process to exit
  time.Sleep(500 * time.Millisecond)

  // Remove target
  err := os.Remove(bakTarget)
  if err != nil {
    FatalError(fmt.Errorf("could not remove old version: %s", err.Error()))
  }

  // Just exit
  os.Exit(0)
}
